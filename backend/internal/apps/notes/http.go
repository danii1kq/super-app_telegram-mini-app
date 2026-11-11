package notes

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
	mux.Handle("GET /api/v1/notes", h.auth.Middleware(http.HandlerFunc(h.list)))
	mux.Handle("POST /api/v1/notes", h.auth.Middleware(http.HandlerFunc(h.create)))
	mux.Handle("GET /api/v1/notes/{id}", h.auth.Middleware(http.HandlerFunc(h.get)))
	mux.Handle("PATCH /api/v1/notes/{id}", h.auth.Middleware(http.HandlerFunc(h.update)))
	mux.Handle("DELETE /api/v1/notes/{id}", h.auth.Middleware(http.HandlerFunc(h.remove)))
	mux.Handle("POST /api/v1/notes/{id}/grants", h.auth.Middleware(http.HandlerFunc(h.grant)))
	mux.Handle("DELETE /api/v1/notes/{id}/grants/{recipient}", h.auth.Middleware(http.HandlerFunc(h.revoke)))
}

type noteDTO struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Body      string `json:"body,omitempty"`
	Version   int    `json:"version"`
	Mine      bool   `json:"mine"`
	UpdatedAt string `json:"updated_at"`
}

func toDTO(note Note, withBody bool) noteDTO {
	dto := noteDTO{
		ID:        strconv.FormatInt(note.ID, 10),
		Title:     note.Title,
		Version:   note.Version,
		Mine:      note.Mine,
		UpdatedAt: note.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
	}
	if withBody {
		dto.Body = note.Body
	}
	return dto
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFrom(r.Context())
	if !ok {
		httpapi.WriteError(w, httpapi.Unauthorized("Требуется вход"))
		return
	}
	notes, err := h.service.List(r.Context(), user.ID)
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}
	out := make([]noteDTO, 0, len(notes))
	for _, note := range notes {
		out = append(out, toDTO(note, false))
	}
	httpapi.WriteJSON(w, http.StatusOK, map[string]any{"notes": out})
}

type noteRequest struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFrom(r.Context())
	if !ok {
		httpapi.WriteError(w, httpapi.Unauthorized("Требуется вход"))
		return
	}
	var body noteRequest
	if err := httpapi.DecodeJSON(w, r, &body, 64<<10); err != nil {
		httpapi.WriteError(w, err)
		return
	}
	note, err := h.service.Create(r.Context(), user.ID, NoteInput{Title: body.Title, Body: body.Body})
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusCreated, toDTO(note, true))
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFrom(r.Context())
	if !ok {
		httpapi.WriteError(w, httpapi.Unauthorized("Требуется вход"))
		return
	}
	noteID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httpapi.WriteError(w, httpapi.BadRequest("invalid_id", "Некорректный ID"))
		return
	}
	note, err := h.service.Get(r.Context(), user.ID, noteID)
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, toDTO(note, true))
}

type updateRequest struct {
	Title   string `json:"title"`
	Body    string `json:"body"`
	Version int    `json:"version"`
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFrom(r.Context())
	if !ok {
		httpapi.WriteError(w, httpapi.Unauthorized("Требуется вход"))
		return
	}
	noteID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httpapi.WriteError(w, httpapi.BadRequest("invalid_id", "Некорректный ID"))
		return
	}
	var body updateRequest
	if err := httpapi.DecodeJSON(w, r, &body, 64<<10); err != nil {
		httpapi.WriteError(w, err)
		return
	}
	note, err := h.service.Update(r.Context(), user.ID, noteID, NoteInput{Title: body.Title, Body: body.Body}, body.Version)
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, toDTO(note, true))
}

func (h *Handler) remove(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFrom(r.Context())
	if !ok {
		httpapi.WriteError(w, httpapi.Unauthorized("Требуется вход"))
		return
	}
	noteID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httpapi.WriteError(w, httpapi.BadRequest("invalid_id", "Некорректный ID"))
		return
	}
	if err := h.service.Delete(r.Context(), user.ID, noteID); err != nil {
		httpapi.WriteError(w, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

type grantRequest struct {
	Recipients  []string `json:"recipients"`
	ContextKind string   `json:"context_kind"`
	ContextID   string   `json:"context_id"`
}

func (h *Handler) grant(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFrom(r.Context())
	if !ok {
		httpapi.WriteError(w, httpapi.Unauthorized("Требуется вход"))
		return
	}
	noteID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httpapi.WriteError(w, httpapi.BadRequest("invalid_id", "Некорректный ID"))
		return
	}
	var body grantRequest
	if err := httpapi.DecodeJSON(w, r, &body, 16<<10); err != nil {
		httpapi.WriteError(w, err)
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

	var contextID *int64
	if body.ContextID != "" {
		id, err := strconv.ParseInt(body.ContextID, 10, 64)
		if err != nil {
			httpapi.WriteError(w, httpapi.BadRequest("invalid_context", "Некорректный контекст"))
			return
		}
		contextID = &id
	}

	if err := h.service.Grant(r.Context(), user.ID, noteID, recipients, body.ContextKind, contextID); err != nil {
		httpapi.WriteError(w, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *Handler) revoke(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFrom(r.Context())
	if !ok {
		httpapi.WriteError(w, httpapi.Unauthorized("Требуется вход"))
		return
	}
	noteID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httpapi.WriteError(w, httpapi.BadRequest("invalid_id", "Некорректный ID"))
		return
	}
	recipientID, err := strconv.ParseInt(r.PathValue("recipient"), 10, 64)
	if err != nil {
		httpapi.WriteError(w, httpapi.BadRequest("invalid_recipient", "Некорректный получатель"))
		return
	}
	if err := h.service.Revoke(r.Context(), user.ID, noteID, recipientID); err != nil {
		httpapi.WriteError(w, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
