package pulse

import (
	"net/http"
	"strconv"

	"github.com/danii1kq/srunal/internal/core/auth"
	"github.com/danii1kq/srunal/internal/platform/httpapi"
)

func (h *Handler) responders(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFrom(r.Context())
	if !ok {
		httpapi.WriteError(w, httpapi.Unauthorized("Требуется вход"))
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		httpapi.WriteError(w, httpapi.BadRequest("invalid_id", "Некорректный ID"))
		return
	}
	responders, err := h.service.Responders(r.Context(), user.ID, id)
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}
	out := make([]map[string]string, 0, len(responders))
	for _, responder := range responders {
		out = append(out, map[string]string{"id": strconv.FormatInt(responder.ID, 10), "public_name": responder.Name})
	}
	httpapi.WriteJSON(w, http.StatusOK, map[string]any{"people": out})
}

func (h *Handler) cancelResponse(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFrom(r.Context())
	if !ok {
		httpapi.WriteError(w, httpapi.Unauthorized("Требуется вход"))
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		httpapi.WriteError(w, httpapi.BadRequest("invalid_id", "Некорректный ID"))
		return
	}
	if err := h.service.CancelResponse(r.Context(), user.ID, id); err != nil {
		httpapi.WriteError(w, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
