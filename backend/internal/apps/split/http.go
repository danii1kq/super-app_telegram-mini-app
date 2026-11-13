package split

import (
	"net/http"
	"strconv"

	"github.com/danii1kq/srunal/internal/core/auth"
	"github.com/danii1kq/srunal/internal/platform/httpapi"
	"github.com/danii1kq/srunal/internal/platform/postgres"
)

type Handler struct {
	service *Service
	auth    *auth.Service
	pool    *postgres.Pool
}

func NewHandler(service *Service, authService *auth.Service, pool *postgres.Pool) *Handler {
	return &Handler{service: service, auth: authService, pool: pool}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.Handle("GET /api/v1/split/groups", h.auth.Middleware(http.HandlerFunc(h.listGroups)))
	mux.Handle("POST /api/v1/split/groups", h.auth.Middleware(http.HandlerFunc(h.createGroup)))
	mux.Handle("GET /api/v1/split/groups/{id}/expenses", h.auth.Middleware(http.HandlerFunc(h.listExpenses)))
	mux.Handle("POST /api/v1/split/groups/{id}/expenses", h.auth.Middleware(http.HandlerFunc(h.addExpense)))
	mux.Handle("GET /api/v1/split/groups/{id}/balances", h.auth.Middleware(http.HandlerFunc(h.balances)))
	mux.Handle("GET /api/v1/split/groups/{id}/members", h.auth.Middleware(http.HandlerFunc(h.members)))
	h.RegisterReceipts(mux)
	h.RegisterReport(mux)
}

func (h *Handler) listGroups(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFrom(r.Context())
	if !ok {
		httpapi.WriteError(w, httpapi.Unauthorized("Требуется вход"))
		return
	}
	groups, err := h.service.ListGroups(r.Context(), user.ID)
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}
	out := make([]map[string]any, 0, len(groups))
	for _, group := range groups {
		out = append(out, map[string]any{
			"id":         strconv.FormatInt(group.ID, 10),
			"title":      group.Title,
			"owner_id":   strconv.FormatInt(group.OwnerID, 10),
			"created_at": group.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
		})
	}
	httpapi.WriteJSON(w, http.StatusOK, map[string]any{"groups": out})
}

type createGroupRequest struct {
	Title   string   `json:"title"`
	Members []string `json:"members"`
}

func (h *Handler) createGroup(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFrom(r.Context())
	if !ok {
		httpapi.WriteError(w, httpapi.Unauthorized("Требуется вход"))
		return
	}
	var body createGroupRequest
	if err := httpapi.DecodeJSON(w, r, &body, 16<<10); err != nil {
		httpapi.WriteError(w, err)
		return
	}

	members, err := parseIDs(body.Members)
	if err != nil {
		httpapi.WriteError(w, httpapi.BadRequest("invalid_member", "Некорректный участник"))
		return
	}

	group, err := h.service.CreateGroup(r.Context(), user.ID, body.Title, members)
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusCreated, map[string]any{
		"id":         strconv.FormatInt(group.ID, 10),
		"title":      group.Title,
		"owner_id":   strconv.FormatInt(group.OwnerID, 10),
		"created_at": group.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
	})
}

func (h *Handler) listExpenses(w http.ResponseWriter, r *http.Request) {
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

	expenses, err := h.service.Expenses(r.Context(), user.ID, groupID)
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}
	out := make([]map[string]any, 0, len(expenses))
	for _, expense := range expenses {
		out = append(out, expenseJSON(expense))
	}
	httpapi.WriteJSON(w, http.StatusOK, map[string]any{"expenses": out})
}

type addExpenseRequest struct {
	ClientRequestID string   `json:"client_request_id"`
	Description     string   `json:"description"`
	AmountMinor     string   `json:"amount_minor"`
	PayerID         string   `json:"payer_id"`
	Participants    []string `json:"participants"`
	Category        string   `json:"category"`
}

func (h *Handler) addExpense(w http.ResponseWriter, r *http.Request) {
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

	var body addExpenseRequest
	if err := httpapi.DecodeJSON(w, r, &body, 16<<10); err != nil {
		httpapi.WriteError(w, err)
		return
	}

	// Amounts travel as strings of minor units: no float, no precision loss.
	amount, err := strconv.ParseInt(body.AmountMinor, 10, 64)
	if err != nil {
		httpapi.WriteError(w, httpapi.Validation(map[string]string{"amount_minor": "Некорректная сумма"}, "Проверьте поля"))
		return
	}
	payerID, err := strconv.ParseInt(body.PayerID, 10, 64)
	if err != nil {
		httpapi.WriteError(w, httpapi.Validation(map[string]string{"payer_id": "Некорректный плательщик"}, "Проверьте поля"))
		return
	}
	participants, err := parseIDs(body.Participants)
	if err != nil {
		httpapi.WriteError(w, httpapi.Validation(map[string]string{"participants": "Некорректный участник"}, "Проверьте поля"))
		return
	}

	expense, err := h.service.AddExpense(r.Context(), user.ID, groupID, payerID,
		body.ClientRequestID, body.Description, body.Category, amount, participants)
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusCreated, expenseJSON(expense))
}

func (h *Handler) balances(w http.ResponseWriter, r *http.Request) {
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

	balances, err := h.service.Balances(r.Context(), user.ID, groupID)
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}
	out := make([]map[string]any, 0, len(balances))
	for _, balance := range balances {
		out = append(out, map[string]any{
			"user_id":       strconv.FormatInt(balance.UserID, 10),
			"paid_minor":    strconv.FormatInt(balance.PaidMinor, 10),
			"share_minor":   strconv.FormatInt(balance.ShareMinor, 10),
			"balance_minor": strconv.FormatInt(balance.BalanceMinor, 10),
			"settled_minor": strconv.FormatInt(balance.SettledMinor, 10),
		})
	}
	httpapi.WriteJSON(w, http.StatusOK, map[string]any{"balances": out})
}

func expenseJSON(expense Expense) map[string]any {
	shares := make(map[string]string, len(expense.Shares))
	for userID, share := range expense.Shares {
		shares[strconv.FormatInt(userID, 10)] = strconv.FormatInt(share, 10)
	}
	return map[string]any{
		"id":           strconv.FormatInt(expense.ID, 10),
		"group_id":     strconv.FormatInt(expense.GroupID, 10),
		"author_id":    strconv.FormatInt(expense.AuthorID, 10),
		"payer_id":     strconv.FormatInt(expense.PayerID, 10),
		"description":  expense.Description,
		"amount_minor": strconv.FormatInt(expense.AmountMinor, 10),
		"spent_at":     expense.SpentAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
		"version":      expense.Version,
		"shares":       shares,
	}
}

func parseIDs(raw []string) ([]int64, error) {
	out := make([]int64, 0, len(raw))
	for _, value := range raw {
		id, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, nil
}

// members lists the group's active members for the pickers.
func (h *Handler) members(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFrom(r.Context())
	if !ok {
		httpapi.WriteError(w, httpapi.Unauthorized("????????? ????"))
		return
	}
	groupID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httpapi.WriteError(w, httpapi.BadRequest("invalid_id", "???????????? ID"))
		return
	}
	ids, err := h.service.Members(r.Context(), user.ID, groupID)
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		out = append(out, strconv.FormatInt(id, 10))
	}
	httpapi.WriteJSON(w, http.StatusOK, map[string]any{"members": out})
}
