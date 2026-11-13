package split

import (
	"net/http"
	"strconv"
	"time"

	"github.com/danii1kq/srunal/internal/core/auth"
	"github.com/danii1kq/srunal/internal/platform/httpapi"
	"github.com/danii1kq/srunal/internal/platform/telegram"
)

func (h *Handler) registerCollaboration(mux *http.ServeMux) {
	mux.Handle("GET /api/v1/split/groups/{id}/receipts", h.auth.Middleware(http.HandlerFunc(h.listReceipts)))
	for _, action := range []string{"collect", "reopen", "cancel", "submit", "respond", "claim", "invite", "revoke-invites"} {
		mux.Handle("POST /api/v1/split/receipts/{id}/"+action, h.auth.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { h.collaborate(w, r, action) })))
	}
	mux.Handle("GET /api/v1/split/invites/{token}", h.auth.Middleware(http.HandlerFunc(h.receiptInvite)))
	mux.Handle("POST /api/v1/split/invites/{token}/join", h.auth.Middleware(http.HandlerFunc(h.receiptInvite)))
}

func (h *Handler) listReceipts(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFrom(r.Context())
	gid, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httpapi.WriteError(w, httpapi.BadRequest("invalid_id", "Некорректная группа"))
		return
	}
	if err = h.service.requireMember(r.Context(), user.ID, gid); err != nil {
		httpapi.WriteError(w, err)
		return
	}

	rows, err := h.pool.Query(r.Context(), `SELECT id,status,version,total_sealed,total_key_id,created_at FROM split_receipts WHERE group_id=$1 ORDER BY updated_at DESC,id DESC LIMIT 100`, gid)
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id int64
		var status string
		var version int
		var sealed []byte
		var key *string
		var created time.Time
		if err = rows.Scan(&id, &status, &version, &sealed, &key, &created); err != nil {
			httpapi.WriteError(w, err)
			return
		}
		total, err := h.service.openOptional(sealed, key, receiptTotalAAD(id))
		if err != nil {
			httpapi.WriteError(w, err)
			return
		}
		out = append(out, map[string]any{"id": strconv.FormatInt(id, 10), "group_id": strconv.FormatInt(gid, 10), "status": status, "version": version, "total_minor": strconv.FormatInt(total, 10), "created_at": created.UTC().Format(time.RFC3339)})
	}
	if err = rows.Err(); err != nil {
		httpapi.WriteError(w, err)
		return
	}

	httpapi.WriteJSON(w, 200, map[string]any{"receipts": out})
}
func (h *Handler) collaborate(w http.ResponseWriter, r *http.Request, action string) {
	user, _ := auth.UserFrom(r.Context())
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httpapi.WriteError(w, httpapi.BadRequest("invalid_id", "Некорректный чек"))
		return
	}
	var body struct {
		Version         int    `json:"version"`
		ItemID          string `json:"item_id"`
		Count           int    `json:"count"`
		Decision        string `json:"decision"`
		Question        string `json:"question"`
		PayerID         string `json:"payer_id"`
		ClientRequestID string `json:"client_request_id"`
		Description     string `json:"description"`
		Category        string `json:"category"`
	}
	if err = httpapi.DecodeJSON(w, r, &body, 8<<10); err != nil {
		httpapi.WriteError(w, err)
		return
	}
	var receipt Receipt
	switch action {
	case "invite":
		var token string
		token, err = h.service.CreateReceiptInvite(r.Context(), user.ID, id)
		if err == nil {
			httpapi.WriteJSON(w, 201, map[string]any{"token": token, "url": telegram.DeepLink(h.service.MiniAppURL, "split_join_"+token)})
			return
		}
	case "revoke-invites":
		err = h.service.RevokeReceiptInvites(r.Context(), user.ID, id)
		if err == nil {
			httpapi.WriteJSON(w, 200, map[string]bool{"revoked": true})
			return
		}
	case "claim":
		var itemID int64
		itemID, err = strconv.ParseInt(body.ItemID, 10, 64)
		if err == nil {
			receipt, err = h.service.ClaimReceiptItem(r.Context(), user.ID, id, itemID, body.Version, body.Count)
		} else {
			err = httpapi.BadRequest("invalid_item", "Некорректная позиция")
		}
	case "submit":
		var payer int64
		payer, err = strconv.ParseInt(body.PayerID, 10, 64)
		if err == nil {
			receipt, err = h.service.SubmitReceipt(r.Context(), user.ID, id, body.Version, ReceiptReview{PayerID: payer, ClientRequestID: body.ClientRequestID, Description: body.Description, Category: body.Category})
		} else {
			err = httpapi.BadRequest("invalid_payer", "Некорректный плательщик")
		}
	case "respond":
		receipt, err = h.service.RespondReceipt(r.Context(), user.ID, id, body.Version, body.Decision, body.Question)
	default:
		receipt, err = h.service.ReceiptAction(r.Context(), user.ID, id, body.Version, action)
	}
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}
	httpapi.WriteJSON(w, 200, receiptJSON(receipt))
}
func (h *Handler) receiptInvite(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFrom(r.Context())
	invite, err := h.service.ResolveReceiptInvite(r.Context(), user.ID, r.PathValue("token"), r.Method == "POST")
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}
	httpapi.WriteJSON(w, 200, invite)
}
