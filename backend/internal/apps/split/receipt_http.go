package split

import (
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/danii1kq/srunal/internal/core/auth"
	"github.com/danii1kq/srunal/internal/platform/httpapi"
)

// RegisterReceipts mounts the receipt and debt routes.
func (h *Handler) RegisterReceipts(mux *http.ServeMux) {
	mux.Handle("POST /api/v1/split/groups/{id}/receipts", h.auth.Middleware(http.HandlerFunc(h.createReceipt)))
	mux.Handle("POST /api/v1/split/groups/{id}/receipts/manual", h.auth.Middleware(http.HandlerFunc(h.createManualReceipt)))
	mux.Handle("GET /api/v1/split/receipts/{id}", h.auth.Middleware(http.HandlerFunc(h.getReceipt)))
	mux.Handle("PATCH /api/v1/split/receipts/{id}", h.auth.Middleware(http.HandlerFunc(h.updateReceipt)))
	mux.Handle("GET /api/v1/split/receipts/{id}/image", h.auth.Middleware(http.HandlerFunc(h.receiptImage)))
	mux.Handle("GET /api/v1/split/receipts/{id}/allocation", h.auth.Middleware(http.HandlerFunc(h.receiptAllocation)))
	mux.Handle("POST /api/v1/split/receipts/{id}/save", h.auth.Middleware(http.HandlerFunc(h.saveReceipt)))

	mux.Handle("GET /api/v1/split/groups/{id}/debts", h.auth.Middleware(http.HandlerFunc(h.listDebts)))
	mux.Handle("POST /api/v1/split/groups/{id}/debts", h.auth.Middleware(http.HandlerFunc(h.createDebt)))
	mux.Handle("POST /api/v1/split/groups/{id}/expenses/{expenseId}/debt", h.auth.Middleware(http.HandlerFunc(h.recordExpenseDebt)))
	mux.Handle("POST /api/v1/split/debts/{id}/payments", h.auth.Middleware(http.HandlerFunc(h.addPayment)))
}

// createReceipt takes a raw image body. Multipart would work too, but one
// binary body keeps the handler small and the size limit exact.
func (h *Handler) createReceipt(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFrom(r.Context())
	if !ok {
		httpapi.WriteError(w, httpapi.Unauthorized("Требуется вход"))
		return
	}
	groupID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || groupID <= 0 {
		httpapi.WriteError(w, httpapi.BadRequest("invalid_id", "Некорректный ID"))
		return
	}
	if err := h.service.requireMember(r.Context(), user.ID, groupID); err != nil {
		httpapi.WriteError(w, err)
		return
	}
	if r.Header.Get("X-AI-Consent") != "deepseek" {
		httpapi.WriteError(w, httpapi.BadRequest("consent_required", "Подтвердите передачу фото в DeepSeek для распознавания"))
		return
	}
	select {
	case h.service.receiptSlots <- struct{}{}:
		defer func() { <-h.service.receiptSlots }()
	default:
		httpapi.WriteError(w, &httpapi.Error{Status: http.StatusTooManyRequests, Code: "receipt_busy", Message: "Распознавание занято, попробуйте ещё раз"})
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxImageBytes+1)

	// One byte over the limit is enough to reject: the service checks the size
	// again, but reading a huge body first would waste memory.
	image, err := io.ReadAll(io.LimitReader(r.Body, maxImageBytes+1))
	if err != nil {
		httpapi.WriteError(w, httpapi.BadRequest("unreadable_body", "Не удалось прочитать файл"))
		return
	}

	receipt, err := h.service.CreateReceipt(r.Context(), user.ID, groupID, image, r.Header.Get("Content-Type"))
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusCreated, receiptJSON(receipt))
}

func (h *Handler) getReceipt(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFrom(r.Context())
	if !ok {
		httpapi.WriteError(w, httpapi.Unauthorized("Требуется вход"))
		return
	}
	receiptID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httpapi.WriteError(w, httpapi.BadRequest("invalid_id", "Некорректный ID"))
		return
	}
	receipt, err := h.service.Receipt(r.Context(), user.ID, receiptID)
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, receiptJSON(receipt))
}

type updateReceiptRequest struct {
	TotalMinor    string              `json:"total_minor"`
	DiscountMinor string              `json:"discount_minor"`
	ServiceMinor  string              `json:"service_minor"`
	Items         []updateReceiptItem `json:"items"`
}

type updateReceiptItem struct {
	Name        string         `json:"name"`
	Quantity    int            `json:"quantity"`
	AmountMinor string         `json:"amount_minor"`
	Unclear     bool           `json:"unclear"`
	Units       map[string]int `json:"units"`
}

func (h *Handler) updateReceipt(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFrom(r.Context())
	if !ok {
		httpapi.WriteError(w, httpapi.Unauthorized("Требуется вход"))
		return
	}
	receiptID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httpapi.WriteError(w, httpapi.BadRequest("invalid_id", "Некорректный ID"))
		return
	}

	var body updateReceiptRequest
	if err := httpapi.DecodeJSON(w, r, &body, 256<<10); err != nil {
		httpapi.WriteError(w, err)
		return
	}

	total, err := parseMinorField(body.TotalMinor, "total_minor")
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}
	discount, err := parseMinorField(body.DiscountMinor, "discount_minor")
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}
	service, err := parseMinorField(body.ServiceMinor, "service_minor")
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}

	items := make([]ReceiptItem, 0, len(body.Items))
	for _, item := range body.Items {
		amount, err := parseMinorField(item.AmountMinor, "items")
		if err != nil {
			httpapi.WriteError(w, err)
			return
		}
		units := make(map[int64]int, len(item.Units))
		for rawID, count := range item.Units {
			userID, err := strconv.ParseInt(rawID, 10, 64)
			if err != nil {
				httpapi.WriteError(w, httpapi.Validation(map[string]string{"items": "Некорректный участник"}, "Проверьте поля"))
				return
			}
			units[userID] = count
		}
		items = append(items, ReceiptItem{
			Name:        item.Name,
			Quantity:    item.Quantity,
			AmountMinor: amount,
			Unclear:     item.Unclear,
			Units:       units,
		})
	}

	receipt, err := h.service.UpdateReceipt(r.Context(), user.ID, receiptID, total, discount, service, items)
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, receiptJSON(receipt))
}

func (h *Handler) receiptImage(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFrom(r.Context())
	if !ok {
		httpapi.WriteError(w, httpapi.Unauthorized("Требуется вход"))
		return
	}
	receiptID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httpapi.WriteError(w, httpapi.BadRequest("invalid_id", "Некорректный ID"))
		return
	}
	image, mime, err := h.service.ReceiptImage(r.Context(), user.ID, receiptID)
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}
	w.Header().Set("Content-Type", mime)
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	_, _ = w.Write(image)
}

func (h *Handler) receiptAllocation(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFrom(r.Context())
	if !ok {
		httpapi.WriteError(w, httpapi.Unauthorized("Требуется вход"))
		return
	}
	receiptID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httpapi.WriteError(w, httpapi.BadRequest("invalid_id", "Некорректный ID"))
		return
	}
	receipt, err := h.service.Receipt(r.Context(), user.ID, receiptID)
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}
	payerID, err := strconv.ParseInt(r.URL.Query().Get("payer_id"), 10, 64)
	if err != nil {
		httpapi.WriteError(w, httpapi.BadRequest("invalid_payer", "Укажите плательщика"))
		return
	}
	allocation := Allocate(receipt, payerID)
	httpapi.WriteJSON(w, http.StatusOK, allocationJSON(allocation))
}

type saveReceiptRequest struct {
	PayerID         string `json:"payer_id"`
	ClientRequestID string `json:"client_request_id"`
	Description     string `json:"description"`
	Category        string `json:"category"`
}

// saveReceipt turns the checked draft into one expense with item-level shares.
func (h *Handler) saveReceipt(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFrom(r.Context())
	if !ok {
		httpapi.WriteError(w, httpapi.Unauthorized("Требуется вход"))
		return
	}
	receiptID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httpapi.WriteError(w, httpapi.BadRequest("invalid_id", "Некорректный ID"))
		return
	}
	var body saveReceiptRequest
	if err := httpapi.DecodeJSON(w, r, &body, 8<<10); err != nil {
		httpapi.WriteError(w, err)
		return
	}
	payerID, err := strconv.ParseInt(body.PayerID, 10, 64)
	if err != nil {
		httpapi.WriteError(w, httpapi.Validation(map[string]string{"payer_id": "Некорректный плательщик"}, "Проверьте поля"))
		return
	}

	expense, err := h.service.SaveReceipt(r.Context(), user.ID, receiptID, payerID, body.ClientRequestID, body.Description, body.Category)
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusCreated, expenseJSON(expense))
}

func (h *Handler) listDebts(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFrom(r.Context())
	if !ok {
		httpapi.WriteError(w, httpapi.Unauthorized("Требуется вход"))
		return
	}
	groupID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httpapi.WriteError(w, httpapi.BadRequest("invalid_id", "Некорректный ID"))
		return
	}
	debts, err := h.service.Debts(r.Context(), user.ID, groupID)
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}
	out := make([]map[string]any, 0, len(debts))
	for _, debt := range debts {
		out = append(out, debtJSON(debt))
	}
	httpapi.WriteJSON(w, http.StatusOK, map[string]any{"debts": out})
}

type createDebtRequest struct {
	ClientRequestID string `json:"client_request_id"`
	CreditorID      string `json:"creditor_id"`
	DebtorID        string `json:"debtor_id"`
	AmountMinor     string `json:"amount_minor"`
	Note            string `json:"note"`
}

func (h *Handler) createDebt(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFrom(r.Context())
	if !ok {
		httpapi.WriteError(w, httpapi.Unauthorized("Требуется вход"))
		return
	}
	groupID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httpapi.WriteError(w, httpapi.BadRequest("invalid_id", "Некорректный ID"))
		return
	}
	var body createDebtRequest
	if err := httpapi.DecodeJSON(w, r, &body, 16<<10); err != nil {
		httpapi.WriteError(w, err)
		return
	}
	creditorID, err := strconv.ParseInt(body.CreditorID, 10, 64)
	if err != nil {
		httpapi.WriteError(w, httpapi.Validation(map[string]string{"creditor_id": "Некорректный получатель"}, "Проверьте поля"))
		return
	}
	debtorID, err := strconv.ParseInt(body.DebtorID, 10, 64)
	if err != nil {
		httpapi.WriteError(w, httpapi.Validation(map[string]string{"debtor_id": "Некорректный должник"}, "Проверьте поля"))
		return
	}
	amount, err := parseMinorField(body.AmountMinor, "amount_minor")
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}

	debt, err := h.service.CreateDebt(r.Context(), user.ID, groupID, creditorID, debtorID,
		amount, body.Note, body.ClientRequestID)
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusCreated, debtJSON(debt))
}

type recordExpenseDebtRequest struct {
	ClientRequestID string `json:"client_request_id"`
	DebtorID        string `json:"debtor_id"`
}

func (h *Handler) recordExpenseDebt(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFrom(r.Context())
	if !ok {
		httpapi.WriteError(w, httpapi.Unauthorized("Требуется вход"))
		return
	}
	groupID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httpapi.WriteError(w, httpapi.BadRequest("invalid_id", "Некорректный ID"))
		return
	}
	expenseID, err := strconv.ParseInt(r.PathValue("expenseId"), 10, 64)
	if err != nil {
		httpapi.WriteError(w, httpapi.BadRequest("invalid_expense", "Некорректный расход"))
		return
	}
	var body recordExpenseDebtRequest
	if err := httpapi.DecodeJSON(w, r, &body, 8<<10); err != nil {
		httpapi.WriteError(w, err)
		return
	}
	debtorID, err := strconv.ParseInt(body.DebtorID, 10, 64)
	if err != nil {
		httpapi.WriteError(w, httpapi.Validation(map[string]string{"debtor_id": "Некорректный должник"}, "Проверьте поля"))
		return
	}

	debt, err := h.service.RecordExpenseDebt(r.Context(), user.ID, groupID, expenseID, debtorID, body.ClientRequestID)
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusCreated, debtJSON(debt))
}

type addPaymentRequest struct {
	ClientRequestID string `json:"client_request_id"`
	AmountMinor     string `json:"amount_minor"`
}

func (h *Handler) addPayment(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFrom(r.Context())
	if !ok {
		httpapi.WriteError(w, httpapi.Unauthorized("Требуется вход"))
		return
	}
	debtID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httpapi.WriteError(w, httpapi.BadRequest("invalid_id", "Некорректный ID"))
		return
	}
	var body addPaymentRequest
	if err := httpapi.DecodeJSON(w, r, &body, 8<<10); err != nil {
		httpapi.WriteError(w, err)
		return
	}
	amount, err := parseMinorField(body.AmountMinor, "amount_minor")
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}

	debt, err := h.service.AddPayment(r.Context(), user.ID, debtID, amount, body.ClientRequestID)
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusCreated, debtJSON(debt))
}

// parseMinorField reads a money field that travels as a string of minor units.
func parseMinorField(raw, field string) (int64, error) {
	if raw == "" {
		return 0, nil
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value < 0 {
		return 0, httpapi.Validation(map[string]string{field: "Некорректная сумма"}, "Проверьте поля")
	}
	return value, nil
}

func receiptJSON(receipt Receipt) map[string]any {
	items := make([]map[string]any, 0, len(receipt.Items))
	for _, item := range receipt.Items {
		units := make(map[string]int, len(item.Units))
		for userID, count := range item.Units {
			units[strconv.FormatInt(userID, 10)] = count
		}
		items = append(items, map[string]any{
			"id":           strconv.FormatInt(item.ID, 10),
			"position":     item.Position,
			"name":         item.Name,
			"quantity":     item.Quantity,
			"amount_minor": strconv.FormatInt(item.AmountMinor, 10),
			"unclear":      item.Unclear,
			"units":        units,
		})
	}

	out := map[string]any{
		"id":             strconv.FormatInt(receipt.ID, 10),
		"group_id":       strconv.FormatInt(receipt.GroupID, 10),
		"author_id":      strconv.FormatInt(receipt.AuthorID, 10),
		"status":         receipt.Status,
		"total_minor":    strconv.FormatInt(receipt.TotalMinor, 10),
		"discount_minor": strconv.FormatInt(receipt.Discount, 10),
		"service_minor":  strconv.FormatInt(receipt.Service, 10),
		"items":          items,
		"created_at":     receipt.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
	}
	out["image_state"], out["image_width"], out["image_height"], out["image_bytes"] = receipt.ImageState, receipt.ImageWidth, receipt.ImageHeight, receipt.ImageBytes
	if receipt.ImageExpiresAt != nil {
		out["image_expires_at"] = receipt.ImageExpiresAt.UTC().Format(time.RFC3339)
	}
	if receipt.OCRError != "" {
		out["ocr_error"] = receipt.OCRError
	}
	if receipt.SavedExpenseID != nil {
		out["saved_expense_id"] = strconv.FormatInt(*receipt.SavedExpenseID, 10)
	}
	return out
}

func allocationJSON(allocation ReceiptAllocation) map[string]any {
	perPerson := make(map[string]string, len(allocation.PerPerson))
	for userID, share := range allocation.PerPerson {
		perPerson[strconv.FormatInt(userID, 10)] = strconv.FormatInt(share, 10)
	}
	owes := make(map[string]string, len(allocation.OwesToPayer))
	for userID, share := range allocation.OwesToPayer {
		owes[strconv.FormatInt(userID, 10)] = strconv.FormatInt(share, 10)
	}
	unallocated := make([]string, 0, len(allocation.Unallocated))
	for _, id := range allocation.Unallocated {
		unallocated = append(unallocated, strconv.FormatInt(id, 10))
	}
	return map[string]any{
		"total_minor":    strconv.FormatInt(allocation.TotalMinor, 10),
		"discount_minor": strconv.FormatInt(allocation.DiscountMinor, 10),
		"service_minor":  strconv.FormatInt(allocation.ServiceMinor, 10),
		"per_person":     perPerson,
		"owes_to_payer":  owes,
		"sum_shares":     strconv.FormatInt(allocation.SumShares, 10),
		"difference":     strconv.FormatInt(allocation.Difference, 10),
		"unallocated":    unallocated,
	}
}

func debtJSON(debt Debt) map[string]any {
	payments := make([]map[string]any, 0, len(debt.Payments))
	for _, payment := range debt.Payments {
		payments = append(payments, map[string]any{
			"id":           strconv.FormatInt(payment.ID, 10),
			"amount_minor": strconv.FormatInt(payment.AmountMinor, 10),
			"paid_at":      payment.PaidAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
			"author_id":    strconv.FormatInt(payment.AuthorID, 10),
		})
	}

	out := map[string]any{
		"id":                strconv.FormatInt(debt.ID, 10),
		"group_id":          strconv.FormatInt(debt.GroupID, 10),
		"author_id":         strconv.FormatInt(debt.AuthorID, 10),
		"creditor_id":       strconv.FormatInt(debt.CreditorID, 10),
		"debtor_id":         strconv.FormatInt(debt.DebtorID, 10),
		"amount_minor":      strconv.FormatInt(debt.AmountMinor, 10),
		"paid_minor":        strconv.FormatInt(debt.PaidMinor, 10),
		"outstanding_minor": strconv.FormatInt(debt.Outstanding, 10),
		"note":              debt.Note,
		"payments":          payments,
		"created_at":        debt.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
	}
	if debt.SourceExpenseID != nil {
		out["source_expense_id"] = strconv.FormatInt(*debt.SourceExpenseID, 10)
	}
	return out
}

// createManualReceipt starts an empty draft for a receipt typed by hand.
func (h *Handler) createManualReceipt(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFrom(r.Context())
	if !ok {
		httpapi.WriteError(w, httpapi.Unauthorized("Требуется вход"))
		return
	}
	groupID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httpapi.WriteError(w, httpapi.BadRequest("invalid_id", "Некорректный ID"))
		return
	}
	receipt, err := h.service.CreateManualReceipt(r.Context(), user.ID, groupID)
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusCreated, receiptJSON(receipt))
}
