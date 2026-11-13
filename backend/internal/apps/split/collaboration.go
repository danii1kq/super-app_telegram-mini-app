package split

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/danii1kq/srunal/internal/platform/httpapi"
	"github.com/danii1kq/srunal/internal/platform/secure"
	"github.com/jackc/pgx/v5"
)

type ReceiptReview struct {
	PayerID         int64  `json:"payer_id"`
	ClientRequestID string `json:"client_request_id"`
	Description     string `json:"description"`
	Category        string `json:"category"`
}
type ReceiptApproval struct {
	UserID   int64
	Decision string
	Question string
}

func lockGroup(ctx context.Context, tx pgx.Tx, groupID int64) error {
	var id int64
	err := tx.QueryRow(ctx, `SELECT id FROM split_groups WHERE id=$1 AND archived_at IS NULL FOR UPDATE`, groupID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return httpapi.NotFound("Группа не найдена")
	}
	return err
}

func (s *Service) lockedReceipt(ctx context.Context, tx pgx.Tx, userID, receiptID int64, version int) (Receipt, error) {
	var groupID int64
	if err := tx.QueryRow(ctx, `SELECT group_id FROM split_receipts WHERE id=$1`, receiptID).Scan(&groupID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Receipt{}, httpapi.NotFound("Чек не найден")
		}
		return Receipt{}, err
	}
	if err := lockGroup(ctx, tx, groupID); err != nil {
		return Receipt{}, err
	}
	if _, err := tx.Exec(ctx, `SELECT id FROM split_receipts WHERE id=$1 FOR UPDATE`, receiptID); err != nil {
		return Receipt{}, err
	}
	receipt, err := s.receipt(ctx, tx, userID, receiptID)
	if err != nil {
		return Receipt{}, err
	}
	if version < 1 || receipt.Version != version {
		return Receipt{}, httpapi.Conflict("version_conflict", "Чек изменился. Обновите его и повторите действие")
	}
	return receipt, nil
}

func (s *Service) ReceiptAction(ctx context.Context, userID, receiptID int64, version int, action string) (Receipt, error) {
	err := s.pool.InTx(ctx, func(tx pgx.Tx) error {
		receipt, err := s.lockedReceipt(ctx, tx, userID, receiptID, version)
		if err != nil {
			return err
		}
		if receipt.AuthorID != userID {
			return httpapi.Forbidden("Действие доступно автору чека")
		}
		if receipt.Status == "saved" || receipt.Status == "cancelled" {
			return httpapi.Conflict("receipt_locked", "Чек уже завершён")
		}
		status := ""
		switch action {
		case "collect":
			if receipt.Status != "draft" {
				return httpapi.Conflict("receipt_locked", "Сначала верните чек к редактированию")
			}
			if err = checkedReceipt(receipt, false); err != nil {
				return err
			}
			status = "collecting"
		case "reopen":
			status = "draft"
		case "cancel":
			status = "cancelled"
		default:
			return httpapi.BadRequest("invalid_action", "Некорректное действие")
		}
		if _, err = tx.Exec(ctx, `DELETE FROM split_receipt_approvals WHERE receipt_id=$1`, receiptID); err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `UPDATE split_receipts SET status=$2,version=version+1,review_sealed=NULL,review_key_id=NULL,updated_at=now() WHERE id=$1`, receiptID, status)
		return err
	})
	if err != nil {
		return Receipt{}, err
	}
	return s.Receipt(ctx, userID, receiptID)
}

func checkedReceipt(r Receipt, allocated bool) error {
	if len(r.Items) == 0 || r.TotalMinor <= 0 {
		return httpapi.BadRequest("invalid_receipt", "Проверьте позиции и итог чека")
	}
	var sum int64
	for _, item := range r.Items {
		if item.Unclear || strings.TrimSpace(item.Name) == "" {
			return httpapi.Validation(map[string]string{"items": "Проверьте каждую распознанную позицию"}, "Проверьте чек")
		}
		sum += item.AmountMinor
	}
	if sum+r.Service-r.Discount != r.TotalMinor {
		return httpapi.Conflict("receipt_difference", "Сумма позиций не совпадает с итогом")
	}
	if allocated {
		a := Allocate(r, 0)
		if len(a.Unallocated) > 0 || a.Difference != 0 || !CheckShares(r.TotalMinor, a.PerPerson) {
			return httpapi.Conflict("receipt_unallocated", "Распределите все позиции")
		}
		for _, item := range r.Items {
			if item.ParentPosition == nil && item.AllocationMode == "units" {
				var n int
				for _, u := range item.Units {
					n += u
				}
				if n != item.Quantity {
					return httpapi.Conflict("receipt_unallocated", "Распределите все единицы товара")
				}
			}
		}
	}
	return nil
}

func (s *Service) ClaimReceiptItem(ctx context.Context, userID, receiptID, itemID int64, version, count int) (Receipt, error) {
	err := s.pool.InTx(ctx, func(tx pgx.Tx) error {
		r, err := s.lockedReceipt(ctx, tx, userID, receiptID, version)
		if err != nil {
			return err
		}
		if r.Status != "collecting" {
			return httpapi.Conflict("receipt_locked", "Сбор выбора сейчас закрыт")
		}
		found := -1
		for i, item := range r.Items {
			if item.ID == itemID {
				found = i
				break
			}
		}
		if found < 0 {
			return httpapi.NotFound("Позиция не найдена")
		}
		item := &r.Items[found]
		if item.ParentPosition != nil {
			return httpapi.BadRequest("combo_child", "Выберите основную позицию комбо")
		}
		if count < 0 || count > 999 || item.AllocationMode == "shared" && count > 1 {
			return httpapi.BadRequest("invalid_units", "Некорректное количество")
		}
		if count == 0 {
			delete(item.Units, userID)
		} else {
			item.Units[userID] = count
		}
		if err = normalizeReceiptItems(r.Items); err != nil {
			return err
		}
		for _, line := range r.Items {
			if line.ID != itemID && (line.ParentPosition == nil || *line.ParentPosition != found) {
				continue
			}
			if _, err = tx.Exec(ctx, `DELETE FROM split_receipt_item_units WHERE item_id=$1 AND user_id=$2`, line.ID, userID); err != nil {
				return err
			}
			if count > 0 {
				if _, err = tx.Exec(ctx, `INSERT INTO split_receipt_item_units(item_id,user_id,units)VALUES($1,$2,$3)`, line.ID, userID, count); err != nil {
					return err
				}
			}
		}
		_, err = tx.Exec(ctx, `UPDATE split_receipts SET version=version+1,updated_at=now() WHERE id=$1`, receiptID)
		return err
	})
	if err != nil {
		return Receipt{}, err
	}
	return s.Receipt(ctx, userID, receiptID)
}

func (s *Service) SubmitReceipt(ctx context.Context, userID, receiptID int64, version int, review ReceiptReview) (Receipt, error) {
	review.Description = strings.TrimSpace(review.Description)
	if review.Description == "" {
		review.Description = "Чек"
	}
	review.Category = normalizeCategory(review.Category)
	if review.PayerID <= 0 || review.ClientRequestID == "" || len(review.ClientRequestID) > 128 || len([]rune(review.Description)) > 200 {
		return Receipt{}, httpapi.BadRequest("invalid_review", "Проверьте плательщика и описание")
	}
	err := s.pool.InTx(ctx, func(tx pgx.Tx) error {
		r, err := s.lockedReceipt(ctx, tx, userID, receiptID, version)
		if err != nil {
			return err
		}
		if r.AuthorID != userID {
			return httpapi.Forbidden("Действие доступно автору чека")
		}
		if r.Status != "draft" && r.Status != "collecting" {
			return httpapi.Conflict("receipt_locked", "Чек уже отправлен на подтверждение")
		}
		if err = checkedReceipt(r, true); err != nil {
			return err
		}
		users := map[int64]bool{userID: true, review.PayerID: true}
		for _, item := range r.Items {
			for id := range item.Units {
				users[id] = true
			}
		}
		for id := range users {
			var active bool
			if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM split_members WHERE group_id=$1 AND user_id=$2 AND state='active')`, r.GroupID, id).Scan(&active); err != nil {
				return err
			}
			if !active {
				return httpapi.BadRequest("invalid_member", "Участник не входит в группу")
			}
		}
		raw, err := json.Marshal(review)
		if err != nil {
			return err
		}
		sealed, err := s.content.Seal(raw, "split_review:"+strconv.FormatInt(receiptID, 10))
		if err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `DELETE FROM split_receipt_approvals WHERE receipt_id=$1`, receiptID); err != nil {
			return err
		}
		for id := range users {
			decision := "pending"
			if id == userID {
				decision = "accepted"
			}
			if _, err = tx.Exec(ctx, `INSERT INTO split_receipt_approvals(receipt_id,user_id,revision,decision)VALUES($1,$2,$3,$4)`, receiptID, id, r.Version+1, decision); err != nil {
				return err
			}
		}
		if _, err = tx.Exec(ctx, `UPDATE split_receipts SET status='review',version=version+1,review_sealed=$2,review_key_id=$3,updated_at=now() WHERE id=$1`, receiptID, sealed.Ciphertext, sealed.KeyID); err != nil {
			return err
		}
		if len(users) == 1 {
			_, err = s.saveReceiptTx(ctx, tx, userID, receiptID, review.PayerID, review.ClientRequestID, review.Description, review.Category)
		}
		return err
	})
	if err != nil {
		return Receipt{}, err
	}
	return s.Receipt(ctx, userID, receiptID)
}

func (s *Service) RespondReceipt(ctx context.Context, userID, receiptID int64, version int, decision, question string) (Receipt, error) {
	if decision != "accepted" && decision != "question" {
		return Receipt{}, httpapi.BadRequest("invalid_decision", "Некорректный ответ")
	}
	question = strings.TrimSpace(question)
	if len([]rune(question)) > 500 || decision == "question" && question == "" {
		return Receipt{}, httpapi.BadRequest("invalid_question", "Опишите вопрос, до 500 символов")
	}
	err := s.pool.InTx(ctx, func(tx pgx.Tx) error {
		r, err := s.lockedReceipt(ctx, tx, userID, receiptID, version)
		if err != nil {
			return err
		}
		if r.Status != "review" {
			return httpapi.Conflict("receipt_locked", "Чек сейчас не ожидает подтверждения")
		}
		var old string
		if err = tx.QueryRow(ctx, `SELECT decision FROM split_receipt_approvals WHERE receipt_id=$1 AND user_id=$2 AND revision=$3`, receiptID, userID, version).Scan(&old); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return httpapi.Forbidden("У вас нет доли в этом чеке")
			}
			return err
		}
		if old == decision {
			return nil
		}
		if old != "pending" {
			return httpapi.Conflict("already_responded", "Ответ уже записан. Автор может вернуть чек к редактированию")
		}
		var qseal []byte
		var qkey *string
		if decision == "question" {
			sealed, err := s.content.SealString(question, "split_question:"+strconv.FormatInt(receiptID, 10)+":"+strconv.FormatInt(userID, 10))
			if err != nil {
				return err
			}
			qseal = sealed.Ciphertext
			qkey = &sealed.KeyID
		}
		if _, err = tx.Exec(ctx, `UPDATE split_receipt_approvals SET decision=$3,question_sealed=$4,question_key_id=$5 WHERE receipt_id=$1 AND user_id=$2`, receiptID, userID, decision, qseal, qkey); err != nil {
			return err
		}
		var remaining int
		if err = tx.QueryRow(ctx, `SELECT count(*) FROM split_receipt_approvals WHERE receipt_id=$1 AND decision<>'accepted'`, receiptID).Scan(&remaining); err != nil {
			return err
		}
		if remaining == 0 {
			if r.Review == nil {
				return httpapi.Conflict("missing_review", "Верните чек к редактированию")
			}
			review := r.Review
			_, err = s.saveReceiptTx(ctx, tx, r.AuthorID, receiptID, review.PayerID, review.ClientRequestID, review.Description, review.Category)
		}
		return err
	})
	if err != nil {
		return Receipt{}, err
	}
	return s.Receipt(ctx, userID, receiptID)
}

func (s *Service) receiptReview(ctx context.Context, q receiptQuerier, r *Receipt, viewerID int64) error {
	var raw []byte
	var key *string
	if err := q.QueryRow(ctx, `SELECT review_sealed,review_key_id FROM split_receipts WHERE id=$1`, r.ID).Scan(&raw, &key); err != nil {
		return err
	}
	if len(raw) > 0 && key != nil {
		data, err := s.content.Open(secure.Sealed{Ciphertext: raw, KeyID: *key}, "split_review:"+strconv.FormatInt(r.ID, 10))
		if err != nil {
			return err
		}
		r.Review = &ReceiptReview{}
		if err = json.Unmarshal(data, r.Review); err != nil {
			return err
		}
	}
	rows, err := q.Query(ctx, `SELECT user_id,decision,question_sealed,question_key_id FROM split_receipt_approvals WHERE receipt_id=$1 ORDER BY user_id`, r.ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	r.Approvals = []ReceiptApproval{}
	for rows.Next() {
		var a ReceiptApproval
		var sealed []byte
		var key *string
		if err = rows.Scan(&a.UserID, &a.Decision, &sealed, &key); err != nil {
			return err
		}
		if len(sealed) > 0 && key != nil && (viewerID == r.AuthorID || viewerID == a.UserID) {
			a.Question, err = s.content.OpenString(secure.Sealed{Ciphertext: sealed, KeyID: *key}, "split_question:"+strconv.FormatInt(r.ID, 10)+":"+strconv.FormatInt(a.UserID, 10))
			if err != nil {
				return err
			}
		}
		r.Approvals = append(r.Approvals, a)
	}
	return rows.Err()
}

type ReceiptInvite struct {
	GroupTitle string    `json:"group_title"`
	ReceiptID  string    `json:"receipt_id"`
	GroupID    string    `json:"group_id"`
	Joined     bool      `json:"joined"`
	ExpiresAt  time.Time `json:"expires_at"`
}

func (s *Service) CreateReceiptInvite(ctx context.Context, userID, receiptID int64) (string, error) {
	r, err := s.Receipt(ctx, userID, receiptID)
	if err != nil {
		return "", err
	}
	if r.AuthorID != userID {
		return "", httpapi.Forbidden("Ссылку создаёт автор чека")
	}
	if r.Status != "collecting" {
		return "", httpapi.Conflict("receipt_locked", "Сначала откройте сбор выбора")
	}
	token, err := secure.NewToken(32)
	if err != nil {
		return "", err
	}
	_, err = s.pool.Exec(ctx, `INSERT INTO split_receipt_invites(token_hash,receipt_id,author_id)VALUES($1,$2,$3)`, secure.TokenHash(token), receiptID, userID)
	return token, err
}
func (s *Service) RevokeReceiptInvites(ctx context.Context, userID, receiptID int64) error {
	_, err := s.Receipt(ctx, userID, receiptID)
	if err != nil {
		return err
	}
	result, err := s.pool.Exec(ctx, `UPDATE split_receipt_invites SET revoked_at=now() WHERE receipt_id=$1 AND EXISTS(SELECT 1 FROM split_receipts r JOIN split_groups g ON g.id=r.group_id WHERE r.id=$1 AND (r.author_id=$2 OR g.owner_id=$2))`, receiptID, userID)
	if err == nil && result.RowsAffected() == 0 {
		return httpapi.Forbidden("Ссылки отзывает автор или владелец группы")
	}
	return err
}
func (s *Service) ResolveReceiptInvite(ctx context.Context, userID int64, token string, join bool) (ReceiptInvite, error) {
	if len(token) < 32 || len(token) > 64 {
		return ReceiptInvite{}, httpapi.NotFound("Приглашение недействительно")
	}
	var out ReceiptInvite
	err := s.pool.InTx(ctx, func(tx pgx.Tx) error {
		var rid, gid int64
		var title []byte
		var key string
		err := tx.QueryRow(ctx, `SELECT r.id,r.group_id,g.title_sealed,g.key_id,i.expires_at FROM split_receipt_invites i JOIN split_receipts r ON r.id=i.receipt_id JOIN split_groups g ON g.id=r.group_id WHERE i.token_hash=$1 AND i.revoked_at IS NULL AND i.expires_at>now() AND g.archived_at IS NULL AND r.status IN ('collecting','review','saved') FOR SHARE OF i`, secure.TokenHash(token)).Scan(&rid, &gid, &title, &key, &out.ExpiresAt)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return httpapi.NotFound("Приглашение истекло или отозвано")
			}
			return err
		}
		out.GroupTitle, err = s.content.OpenString(secure.Sealed{Ciphertext: title, KeyID: key}, groupTitleAAD(gid))
		if err != nil {
			return err
		}
		if join {
			if _, err = tx.Exec(ctx, `INSERT INTO split_members(group_id,user_id)VALUES($1,$2) ON CONFLICT(group_id,user_id)DO UPDATE SET state='active'`, gid, userID); err != nil {
				return err
			}
		}
		if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM split_members WHERE group_id=$1 AND user_id=$2 AND state='active')`, gid, userID).Scan(&out.Joined); err != nil {
			return err
		}
		if out.Joined {
			out.ReceiptID = strconv.FormatInt(rid, 10)
			out.GroupID = strconv.FormatInt(gid, 10)
		}
		return nil
	})
	return out, err
}
