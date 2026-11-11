package chat

import (
	"net/http"
	"strconv"

	"github.com/danii1kq/srunal/internal/core/auth"
	"github.com/danii1kq/srunal/internal/platform/httpapi"
)

type openDirectRequest struct {
	UserID string `json:"user_id"`
}

// openDirect opens (or reuses) the one-to-one dialog with another account.
func (h *Handler) openDirect(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFrom(r.Context())
	if !ok {
		httpapi.WriteError(w, httpapi.Unauthorized("Требуется вход"))
		return
	}
	var body openDirectRequest
	if err := httpapi.DecodeJSON(w, r, &body, 4<<10); err != nil {
		httpapi.WriteError(w, err)
		return
	}
	otherID, err := strconv.ParseInt(body.UserID, 10, 64)
	if err != nil {
		httpapi.WriteError(w, httpapi.BadRequest("invalid_user", "Некорректный пользователь"))
		return
	}

	conversation, err := h.service.OpenDirect(r.Context(), user.ID, otherID)
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, map[string]any{
		"id":         formatID(conversation.ID),
		"kind":       conversation.Kind,
		"title":      conversation.Title,
		"last_seq":   formatID(conversation.LastSeq),
		"last_read":  "0",
		"unread":     0,
		"updated_at": conversation.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
	})
}
