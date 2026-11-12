package integration

import (
	"net/http"
	"strconv"

	"github.com/danii1kq/srunal/internal/core/auth"
	"github.com/danii1kq/srunal/internal/platform/httpapi"
)

type Handler struct {
	service *Service
	auth    *auth.Service
}

func NewHandler(service *Service, authService *auth.Service) *Handler {
	return &Handler{service: service, auth: authService}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.Handle("POST /api/v1/integrations/doc-deadline", h.auth.Middleware(http.HandlerFunc(h.docDeadline)))
	mux.Handle("POST /api/v1/integrations/note-share", h.auth.Middleware(http.HandlerFunc(h.noteShare)))
}

type docDeadlineRequest struct {
	OrderNumber    string `json:"order_number"`
	Title          string `json:"title"`
	DueDate        string `json:"due_date"`
	IdempotencyKey string `json:"idempotency_key"`
}

func (h *Handler) docDeadline(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFrom(r.Context())
	if !ok {
		httpapi.WriteError(w, httpapi.Unauthorized("Требуется вход"))
		return
	}

	var body docDeadlineRequest
	if err := httpapi.DecodeJSON(w, r, &body, 16<<10); err != nil {
		httpapi.WriteError(w, err)
		return
	}

	initData, err := h.auth.InitDataFor(r.Context(), user.ID, auth.BearerToken(r))
	if err != nil {
		httpapi.WriteError(w, httpapi.Internal(err))
		return
	}

	result, err := h.service.DeadlineFromOrder(r.Context(), user.ID, initData,
		body.IdempotencyKey, body.OrderNumber, body.Title, body.DueDate)
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, result)
}

type noteShareRequest struct {
	NoteID     string   `json:"note_id"`
	Recipients []string `json:"recipients"`
}

func (h *Handler) noteShare(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFrom(r.Context())
	if !ok {
		httpapi.WriteError(w, httpapi.Unauthorized("Требуется вход"))
		return
	}

	var body noteShareRequest
	if err := httpapi.DecodeJSON(w, r, &body, 16<<10); err != nil {
		httpapi.WriteError(w, err)
		return
	}

	noteID, err := strconv.ParseInt(body.NoteID, 10, 64)
	if err != nil {
		httpapi.WriteError(w, httpapi.BadRequest("invalid_note", "Некорректная заметка"))
		return
	}

	recipients := make([]int64, 0, len(body.Recipients))
	for _, raw := range body.Recipients {
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			httpapi.WriteError(w, httpapi.BadRequest("invalid_recipient", "Некорректный получатель"))
			return
		}
		recipients = append(recipients, id)
	}

	card, err := h.service.ShareNoteCard(r.Context(), user.ID, noteID, recipients)
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, card)
}
