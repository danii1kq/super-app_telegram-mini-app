package split

import (
	"context"
	"errors"
	"math"
	"math/big"
	"sort"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/danii1kq/srunal/internal/platform/httpapi"
	"github.com/danii1kq/srunal/internal/platform/media"
	"github.com/danii1kq/srunal/internal/platform/ocr"
	"github.com/danii1kq/srunal/internal/platform/secure"
	"github.com/danii1kq/srunal/internal/platform/vision"
)

// Receipt is a photographed check, from draft to saved expense.
type Receipt struct {
	ID             int64
	GroupID        int64
	AuthorID       int64
	Status         string
	Review         *ReceiptReview
	Approvals      []ReceiptApproval
	Version        int
	OCRError       string
	TotalMinor     int64
	Discount       int64
	Service        int64
	Items          []ReceiptItem
	SavedExpenseID *int64
	CreatedAt      time.Time
	// ImageState is none, stored, purged or processing.
	ImageState string
	// ImageExpiresAt is when the photo becomes eligible for deletion.
	ImageExpiresAt *time.Time
	ImageWidth     int
	ImageHeight    int
	ImageBytes     int
}

// ReceiptItem is one line, with how many units each person takes.
type ReceiptItem struct {
	ID                      int64
	Position                int
	Name                    string
	Quantity                int
	AmountMinor             int64
	Unclear                 bool
	AllocationMode          string
	ParentPosition          *int
	SuggestedParentPosition *int
	// Units maps a user to how many units of this line they take.
	Units map[int64]int
}

// Debt is an obligation between two people.
type Debt struct {
	ID          int64
	GroupID     int64
	AuthorID    int64
	CreditorID  int64
	DebtorID    int64
	AmountMinor int64
	Note        string
	// SourceExpenseID set: the expense already carries this money, so the row
	// must not be added to the balance again.
	SourceExpenseID *int64
	PaidMinor       int64
	Outstanding     int64
	CreatedAt       time.Time
	Payments        []DebtPayment
}

// DebtPayment is one repayment against a debt.
type DebtPayment struct {
	ID          int64
	AmountMinor int64
	PaidAt      time.Time
	AuthorID    int64
}

const maxImageBytes = 20 << 20

// Retention applies only to photos stored by earlier versions.
const savedImageTTL = 30 * 24 * time.Hour

// allowedImageMime is the accepted set. The bytes are sniffed, not trusted from
// the client's content type.
var allowedImageMime = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

// CreateManualReceipt starts an empty draft without a photo.
//
// A receipt typed by hand is a first-class case: recognition may be unavailable
// or the paper may be lost. The row keeps the same shape, with no image.
func (s *Service) CreateManualReceipt(ctx context.Context, authorID, groupID int64) (Receipt, error) {
	if err := s.requireMember(ctx, authorID, groupID); err != nil {
		return Receipt{}, err
	}

	var receipt Receipt
	err := s.pool.InTx(ctx, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx,
			`INSERT INTO split_receipts (group_id, author_id, image_sealed, image_key_id, image_mime, image_bytes, image_state)
			 VALUES ($1, $2, ''::bytea, 'none', 'image/png', 0, 'none')
			 RETURNING id, created_at`,
			groupID, authorID,
		).Scan(&receipt.ID, &receipt.CreatedAt); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return Receipt{}, err
	}

	receipt.GroupID = groupID
	receipt.AuthorID = authorID
	receipt.Status = "draft"
	receipt.Version = 1
	receipt.Items = []ReceiptItem{}
	receipt.ImageState = "none"
	return receipt, nil
}

// CreateReceipt reads a photo in memory and stores only the extracted draft.
//
// Recognition failure leaves a draft for manual entry, never a stored photo.
func (s *Service) CreateReceipt(ctx context.Context, authorID, groupID int64, image []byte, mime string) (Receipt, error) {
	if err := s.requireMember(ctx, authorID, groupID); err != nil {
		return Receipt{}, err
	}
	if len(image) == 0 {
		return Receipt{}, httpapi.Validation(map[string]string{"image": "Файл пустой"}, "Проверьте поля")
	}
	if len(image) > maxImageBytes {
		return Receipt{}, httpapi.Validation(map[string]string{"image": "Файл больше 20 МБ"}, "Проверьте поля")
	}
	if media.DetectMime(image) == "" {
		return Receipt{}, httpapi.Validation(map[string]string{"image": "Поддерживаются JPEG, PNG и WebP"}, "Проверьте поля")
	}

	if err := s.checkReceiptBudget(ctx, s.pool, authorID); err != nil {
		return Receipt{}, err
	}

	// Normalize in memory: photos must not enter PostgreSQL or its backups.
	normalized, err := media.Normalize(image)
	if err != nil {
		switch {
		case errors.Is(err, media.ErrTooLarge):
			return Receipt{}, httpapi.Validation(map[string]string{"image": "Слишком большое разрешение"}, "Проверьте поля")
		default:
			return Receipt{}, httpapi.Validation(map[string]string{"image": "Файл не читается как изображение"}, "Проверьте поля")
		}
	}

	if normalized.StoredBytes > vision.MaxImageBytes {
		return Receipt{}, httpapi.Validation(map[string]string{"image": "После сжатия фото больше 10 МБ: уменьшите разрешение"}, "Проверьте фото")
	}

	var receipt Receipt
	err = s.pool.InTx(ctx, func(tx pgx.Tx) error {
		// Serialize the per-person recognition budget before making a paid request.
		if _, err := tx.Exec(ctx, `SELECT id FROM users WHERE id=$1 FOR UPDATE`, authorID); err != nil {
			return err
		}
		if err := s.checkReceiptBudget(ctx, tx, authorID); err != nil {
			return err
		}

		// Keep the attempt for quotas, without persisting any image bytes.
		if err := tx.QueryRow(ctx,
			`INSERT INTO split_receipts (group_id, author_id, image_sealed, image_key_id, image_mime, image_bytes,
			                             image_state, image_width, image_height)
			 VALUES ($1, $2, ''::bytea, 'purged', $3, 0, 'processing', $4, $5)
			 RETURNING id, created_at`,
			groupID, authorID, normalized.Mime, normalized.Width, normalized.Height,
		).Scan(&receipt.ID, &receipt.CreatedAt); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return Receipt{}, err
	}

	receipt.GroupID = groupID
	receipt.AuthorID = authorID
	receipt.Status = "draft"
	receipt.Version = 1
	receipt.ImageState = "purged"
	receipt.ImageWidth, receipt.ImageHeight = normalized.Width, normalized.Height

	// Recognition reads the normalized image, not the original: the text is the
	// same and the payload is a fraction of the size.
	result, ocrErr := s.reader.Read(ctx, normalized.Data, normalized.Mime)
	if ocrErr != nil {
		code := "unreadable"
		if errors.Is(ocrErr, ocr.ErrUnavailable) {
			code = "unavailable"
		}
		receipt.OCRError = code
		if _, err := s.pool.Exec(ctx,
			`UPDATE split_receipts SET ocr_error = $2, image_state = 'purged', updated_at = now() WHERE id = $1`,
			receipt.ID, code,
		); err != nil {
			return Receipt{}, err
		}
		return receipt, nil
	}

	receipt.TotalMinor = result.Total
	receipt.Discount = result.Discount
	receipt.Service = result.Service
	if err := s.storeRecognized(ctx, receipt.ID, result); err != nil {
		return Receipt{}, err
	}
	receipt.Items, err = s.receiptItems(ctx, s.pool, receipt.ID)
	if err != nil {
		return Receipt{}, err
	}
	return receipt, nil
}

func (s *Service) checkReceiptBudget(ctx context.Context, q receiptQuerier, authorID int64) error {
	var attempts int
	var latest *time.Time
	if err := q.QueryRow(ctx, `SELECT count(*), max(created_at) FROM split_receipts WHERE author_id=$1 AND image_state<>'none' AND created_at>now()-interval '24 hours'`, authorID).Scan(&attempts, &latest); err != nil {
		return err
	}
	if attempts >= 50 || (latest != nil && time.Since(*latest) < 30*time.Second) {
		return &httpapi.Error{Status: 429, Code: "receipt_rate_limited", Message: "Между фото нужно 30 секунд, доступно до 50 распознаваний за 24 часа"}
	}
	return nil
}

// storeRecognized writes the recognized lines and totals onto the draft.
func (s *Service) storeRecognized(ctx context.Context, receiptID int64, result ocr.Result) error {
	return s.pool.InTx(ctx, func(tx pgx.Tx) error {
		for index, item := range result.Items {
			var itemID int64
			if err := tx.QueryRow(ctx,
				`INSERT INTO split_receipt_items (receipt_id, position, name_sealed, name_key_id, quantity, amount_sealed, amount_key_id, confidence)
				 VALUES ($1, $2, ''::bytea, 'pending', $3, ''::bytea, 'pending', $4)
				 RETURNING id`,
				receiptID, index, item.Quantity, confidenceOf(item.Unclear),
			).Scan(&itemID); err != nil {
				return err
			}
			nameSealed, err := s.content.SealString(item.Name, receiptItemNameAAD(itemID))
			if err != nil {
				return err
			}
			amountSealed, err := s.content.SealString(strconv.FormatInt(item.Amount, 10), receiptItemAmountAAD(itemID))
			if err != nil {
				return err
			}
			if _, err := tx.Exec(ctx,
				`UPDATE split_receipt_items
				    SET name_sealed = $2, name_key_id = $3, amount_sealed = $4, amount_key_id = $5, suggested_parent_position=$6
				  WHERE id = $1`,
				itemID, nameSealed.Ciphertext, nameSealed.KeyID, amountSealed.Ciphertext, amountSealed.KeyID, item.ParentPosition,
			); err != nil {
				return err
			}
		}

		totalSealed, totalKey, err := s.sealOptional(result.Total, receiptTotalAAD(receiptID))
		if err != nil {
			return err
		}
		discountSealed, discountKey, err := s.sealOptional(result.Discount, receiptDiscountAAD(receiptID))
		if err != nil {
			return err
		}
		serviceSealed, serviceKey, err := s.sealOptional(result.Service, receiptServiceAAD(receiptID))
		if err != nil {
			return err
		}

		_, err = tx.Exec(ctx,
			`UPDATE split_receipts
			    SET total_sealed = $2, total_key_id = $3,
			        discount_sealed = $4, discount_key_id = $5,
			        service_sealed = $6, service_key_id = $7,
		        ocr_error = NULL, image_state = 'purged', updated_at = now()
			  WHERE id = $1`,
			receiptID, totalSealed, totalKey, discountSealed, discountKey, serviceSealed, serviceKey,
		)
		return err
	})
}

func (s *Service) sealOptional(value int64, aad string) ([]byte, *string, error) {
	if value == 0 {
		return nil, nil, nil
	}
	sealed, err := s.content.SealString(strconv.FormatInt(value, 10), aad)
	if err != nil {
		return nil, nil, err
	}
	return sealed.Ciphertext, &sealed.KeyID, nil
}

// Receipt returns one draft or saved receipt. Only group members may read it.
type receiptQuerier interface {
	QueryRow(context.Context, string, ...any) pgx.Row
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

func (s *Service) Receipt(ctx context.Context, viewerID, receiptID int64) (Receipt, error) {
	return s.receipt(ctx, s.pool, viewerID, receiptID)
}

func (s *Service) receipt(ctx context.Context, q receiptQuerier, viewerID, receiptID int64) (Receipt, error) {
	var receipt Receipt
	var totalSealed, discountSealed, serviceSealed []byte
	var totalKey, discountKey, serviceKey *string
	var ocrError *string
	var savedExpense *int64

	err := q.QueryRow(ctx,
		`SELECT r.id, r.group_id, r.author_id, r.status, r.ocr_error,
		        r.total_sealed, r.total_key_id, r.discount_sealed, r.discount_key_id,
		        r.service_sealed, r.service_key_id, r.saved_expense_id, r.created_at,
            r.image_state, coalesce(r.image_width,0), coalesce(r.image_height,0), r.image_bytes, r.image_expires_at, r.version
		   FROM split_receipts r
		   JOIN split_members m ON m.group_id = r.group_id
		  WHERE r.id = $1 AND m.user_id = $2 AND m.state = 'active'`,
		receiptID, viewerID,
	).Scan(&receipt.ID, &receipt.GroupID, &receipt.AuthorID, &receipt.Status, &ocrError,
		&totalSealed, &totalKey, &discountSealed, &discountKey,
		&serviceSealed, &serviceKey, &savedExpense, &receipt.CreatedAt,
		&receipt.ImageState, &receipt.ImageWidth, &receipt.ImageHeight, &receipt.ImageBytes, &receipt.ImageExpiresAt, &receipt.Version)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Receipt{}, httpapi.NotFound("Чек не найден")
		}
		return Receipt{}, err
	}
	if ocrError != nil {
		receipt.OCRError = *ocrError
	}
	receipt.SavedExpenseID = savedExpense

	if receipt.TotalMinor, err = s.openOptional(totalSealed, totalKey, receiptTotalAAD(receipt.ID)); err != nil {
		return Receipt{}, err
	}
	if receipt.Discount, err = s.openOptional(discountSealed, discountKey, receiptDiscountAAD(receipt.ID)); err != nil {
		return Receipt{}, err
	}
	if receipt.Service, err = s.openOptional(serviceSealed, serviceKey, receiptServiceAAD(receipt.ID)); err != nil {
		return Receipt{}, err
	}

	if err = s.receiptReview(ctx, q, &receipt, viewerID); err != nil {
		return Receipt{}, err
	}
	receipt.Items, err = s.receiptItems(ctx, q, receipt.ID)
	if err != nil {
		return Receipt{}, err
	}
	return receipt, nil
}

func (s *Service) openOptional(sealed []byte, keyID *string, aad string) (int64, error) {
	if len(sealed) == 0 || keyID == nil {
		return 0, nil
	}
	raw, err := s.content.OpenString(secure.Sealed{KeyID: *keyID, Ciphertext: sealed}, aad)
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(raw, 10, 64)
}

func (s *Service) receiptItems(ctx context.Context, q receiptQuerier, receiptID int64) ([]ReceiptItem, error) {
	rows, err := q.Query(ctx,
		`SELECT id, position, name_sealed, name_key_id, quantity, amount_sealed, amount_key_id, confidence, allocation_mode, parent_position, suggested_parent_position
		   FROM split_receipt_items WHERE receipt_id = $1 ORDER BY position`, receiptID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]ReceiptItem, 0, 8)
	for rows.Next() {
		var item ReceiptItem
		var nameSealed, amountSealed []byte
		var nameKey, amountKey, confidence string
		if err := rows.Scan(&item.ID, &item.Position, &nameSealed, &nameKey,
			&item.Quantity, &amountSealed, &amountKey, &confidence, &item.AllocationMode, &item.ParentPosition, &item.SuggestedParentPosition); err != nil {
			return nil, err
		}
		name, err := s.content.OpenString(secure.Sealed{KeyID: nameKey, Ciphertext: nameSealed}, receiptItemNameAAD(item.ID))
		if err != nil {
			return nil, err
		}
		amountRaw, err := s.content.OpenString(secure.Sealed{KeyID: amountKey, Ciphertext: amountSealed}, receiptItemAmountAAD(item.ID))
		if err != nil {
			return nil, err
		}
		amount, err := strconv.ParseInt(amountRaw, 10, 64)
		if err != nil {
			return nil, err
		}
		item.Name = name
		item.AmountMinor = amount
		item.Unclear = confidence == "unclear"
		item.Units = map[int64]int{}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	units, err := s.itemUnits(ctx, q, receiptID)
	if err != nil {
		return nil, err
	}
	for index := range items {
		if found, ok := units[items[index].ID]; ok {
			items[index].Units = found
		}
	}
	return items, nil
}

func (s *Service) itemUnits(ctx context.Context, q receiptQuerier, receiptID int64) (map[int64]map[int64]int, error) {
	rows, err := q.Query(ctx,
		`SELECT u.item_id, u.user_id, u.units
		   FROM split_receipt_item_units u
		   JOIN split_receipt_items i ON i.id = u.item_id
		  WHERE i.receipt_id = $1`, receiptID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := map[int64]map[int64]int{}
	for rows.Next() {
		var itemID, userID int64
		var units int
		if err := rows.Scan(&itemID, &userID, &units); err != nil {
			return nil, err
		}
		if out[itemID] == nil {
			out[itemID] = map[int64]int{}
		}
		out[itemID][userID] = units
	}
	return out, rows.Err()
}

// ReceiptImage returns the sealed photo for a group member.
func (s *Service) ReceiptImage(ctx context.Context, viewerID, receiptID int64) ([]byte, string, error) {
	var sealed []byte
	var keyID, mime string
	err := s.pool.QueryRow(ctx,
		`SELECT r.image_sealed, r.image_key_id, r.image_mime
		   FROM split_receipts r
		   JOIN split_members m ON m.group_id = r.group_id
		  WHERE r.id = $1 AND m.user_id = $2 AND m.state = 'active'
		    AND r.image_state = 'stored' AND octet_length(r.image_sealed) > 0`,
		receiptID, viewerID,
	).Scan(&sealed, &keyID, &mime)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, "", httpapi.NotFound("Чек не найден")
		}
		return nil, "", err
	}
	image, err := s.content.Open(secure.Sealed{KeyID: keyID, Ciphertext: sealed}, receiptImageAAD(receiptID))
	if err != nil {
		return nil, "", err
	}
	return image, mime, nil
}

// ReceiptAllocation is the computed split of one receipt: who owes whom.
type ReceiptAllocation struct {
	TotalMinor    int64
	DiscountMinor int64
	ServiceMinor  int64
	// PerPerson is each person's share of the receipt after discount and service.
	PerPerson map[int64]int64
	// OwesToPayer is what each person must return to the payer.
	OwesToPayer map[int64]int64
	// SumShares must equal TotalMinor; Difference is what is left unallocated.
	SumShares  int64
	Difference int64
	// Unallocated lists lines nobody took, so the UI can point at them.
	Unallocated []int64
}

// Allocate computes the receipt split in Go. The recognizer never decides money.
//
// Discount and service are spread over the people who actually took part, in
// proportion to their share, with the remainder going to the largest shares
// first. Nobody who took nothing pays for a discount or a service fee.
func Allocate(receipt Receipt, payerID int64) ReceiptAllocation {
	perPerson := map[int64]int64{}
	unallocated := make([]int64, 0)

	for _, item := range receipt.Items {
		units := 0
		for _, count := range item.Units {
			units += count
		}
		if units == 0 {
			unallocated = append(unallocated, item.ID)
			continue
		}
		// Split the line by units, not by people: two drinks to one person and
		// one to another must not become an even three-way split.
		byUnits := make(map[int64]int64, len(item.Units))
		for userID, count := range item.Units {
			byUnits[userID] = int64(count)
		}
		shares, err := splitByWeights(item.AmountMinor, byUnits)
		if err != nil {
			continue
		}
		for userID, share := range shares {
			perPerson[userID] += share
		}
	}

	sum := int64(0)
	for _, share := range perPerson {
		sum += share
	}

	// Discount and service apply only to the people who took part.
	adjustment := receipt.Service - receipt.Discount
	if adjustment != 0 && len(perPerson) > 0 {
		weights := make(map[int64]int64, len(perPerson))
		for userID, share := range perPerson {
			if share > 0 {
				weights[userID] = share
			}
		}
		adjusted, err := splitByWeights(adjustment, weights)
		if err == nil {
			for userID, delta := range adjusted {
				perPerson[userID] += delta
			}
		}
	}

	sum = 0
	for _, share := range perPerson {
		sum += share
	}

	owes := map[int64]int64{}
	for userID, share := range perPerson {
		if userID == payerID {
			continue
		}
		if share > 0 {
			owes[userID] = share
		}
	}

	sort.Slice(unallocated, func(i, j int) bool { return unallocated[i] < unallocated[j] })

	return ReceiptAllocation{
		TotalMinor:    receipt.TotalMinor,
		DiscountMinor: receipt.Discount,
		ServiceMinor:  receipt.Service,
		PerPerson:     perPerson,
		OwesToPayer:   owes,
		SumShares:     sum,
		Difference:    receipt.TotalMinor - sum,
		Unallocated:   unallocated,
	}
}

// splitByWeights divides amount by integer weights with a deterministic
// remainder: the largest weights take one extra minor unit each.
func splitByWeights(amount int64, weights map[int64]int64) (map[int64]int64, error) {
	if len(weights) == 0 {
		return nil, ErrNoParticipants
	}
	total := int64(0)
	for _, weight := range weights {
		if weight <= 0 {
			return nil, ErrInvalidAmount
		}
		if total > math.MaxInt64-weight {
			return nil, ErrInvalidAmount
		}
		total += weight
	}
	if total <= 0 {
		return nil, ErrInvalidAmount
	}

	// Stable order: biggest weight first, then ascending id.
	ids := make([]int64, 0, len(weights))
	for id := range weights {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool {
		if weights[ids[i]] != weights[ids[j]] {
			return weights[ids[i]] > weights[ids[j]]
		}
		return ids[i] < ids[j]
	})

	out := make(map[int64]int64, len(ids))
	assigned := int64(0)
	for _, id := range ids {
		// The product can exceed int64 even though the resulting share fits.
		product := new(big.Int).Mul(big.NewInt(amount), big.NewInt(weights[id]))
		share := product.Quo(product, big.NewInt(total)).Int64()
		out[id] = share
		assigned += share
	}
	// Hand out the rounding remainder one minor unit at a time.
	remainder := amount - assigned
	for index := 0; remainder != 0 && index < len(ids)*2; index++ {
		id := ids[index%len(ids)]
		step := int64(1)
		if remainder < 0 {
			step = -1
		}
		out[id] += step
		remainder -= step
	}
	return out, nil
}

func confidenceOf(unclear bool) string {
	if unclear {
		return "unclear"
	}
	return "ok"
}

// detectImageMime sniffs the magic bytes. The client's content type is a hint,
// never the truth.
func detectImageMime(data []byte) string {
	if len(data) < 12 {
		return ""
	}
	switch {
	case data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF:
		return "image/jpeg"
	case string(data[0:8]) == "\x89PNG\r\n\x1a\n":
		return "image/png"
	case string(data[0:4]) == "RIFF" && string(data[8:12]) == "WEBP":
		return "image/webp"
	}
	return ""
}

func receiptImageAAD(receiptID int64) string {
	return "split_receipt_image:" + strconv.FormatInt(receiptID, 10)
}

func receiptTotalAAD(receiptID int64) string {
	return "split_receipt_total:" + strconv.FormatInt(receiptID, 10)
}

func receiptDiscountAAD(receiptID int64) string {
	return "split_receipt_discount:" + strconv.FormatInt(receiptID, 10)
}

func receiptServiceAAD(receiptID int64) string {
	return "split_receipt_service:" + strconv.FormatInt(receiptID, 10)
}

func receiptItemNameAAD(itemID int64) string {
	return "split_receipt_item_name:" + strconv.FormatInt(itemID, 10)
}

func receiptItemAmountAAD(itemID int64) string {
	return "split_receipt_item_amount:" + strconv.FormatInt(itemID, 10)
}
