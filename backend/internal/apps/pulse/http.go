package pulse

import (
	"net/http"
	"strconv"
	"time"

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
	mux.Handle("GET /api/v1/pulse", h.auth.Middleware(http.HandlerFunc(h.feed)))
	mux.Handle("POST /api/v1/pulse", h.auth.Middleware(http.HandlerFunc(h.create)))
	mux.Handle("POST /api/v1/pulse/{id}/responses", h.auth.Middleware(http.HandlerFunc(h.respond)))
	mux.Handle("POST /api/v1/pulse/{id}/close", h.auth.Middleware(http.HandlerFunc(h.close)))
}

func statusJSON(status Status) map[string]any {
	out := map[string]any{
		"id":          strconv.FormatInt(status.ID, 10),
		"author_id":   strconv.FormatInt(status.AuthorID, 10),
		"author_name": status.AuthorName,
		"text":        status.Text,
		"expires_at":  status.ExpiresAt.UTC().Format(time.RFC3339),
		"created_at":  status.CreatedAt.UTC().Format(time.RFC3339),
		"responses":   status.Responses,
		"mine":        status.Mine,
		"responded":   status.Responded,
	}
	if status.ClosedAt != nil {
		out["closed_at"] = status.ClosedAt.UTC().Format(time.RFC3339)
	}
	return out
}

func (h *Handler) feed(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFrom(r.Context())
	if !ok {
		httpapi.WriteError(w, httpapi.Unauthorized("Требуется вход"))
		return
	}
	statuses, err := h.service.Feed(r.Context(), user.ID)
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}
	out := make([]map[string]any, 0, len(statuses))
	for _, status := range statuses {
		out = append(out, statusJSON(status))
	}
	httpapi.WriteJSON(w, http.StatusOK, map[string]any{"statuses": out})
}

type createRequest struct {
	Text        string   `json:"text"`
	ExpiresAt   string   `json:"expires_at"`
	Audience    []string `json:"audience"`
	CommunityID string   `json:"community_id"`
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFrom(r.Context())
	if !ok {
		httpapi.WriteError(w, httpapi.Unauthorized("Требуется вход"))
		return
	}

	var body createRequest
	if err := httpapi.DecodeJSON(w, r, &body, 16<<10); err != nil {
		httpapi.WriteError(w, err)
		return
	}

	expiresAt, err := time.Parse(time.RFC3339, body.ExpiresAt)
	if err != nil {
		httpapi.WriteError(w, httpapi.Validation(map[string]string{"expires_at": "Некорректное время"}, "Проверьте поля"))
		return
	}

	audience := Audience{}
	for _, raw := range body.Audience {
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			httpapi.WriteError(w, httpapi.Validation(map[string]string{"audience": "Некорректный получатель"}, "Проверьте поля"))
			return
		}
		audience.Users = append(audience.Users, id)
	}
	if body.CommunityID != "" {
		id, err := strconv.ParseInt(body.CommunityID, 10, 64)
		if err != nil {
			httpapi.WriteError(w, httpapi.Validation(map[string]string{"community_id": "Некорректное сообщество"}, "Проверьте поля"))
			return
		}
		audience.Community = &id
	}

	status, err := h.service.Create(r.Context(), user.ID, body.Text, expiresAt, audience)
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusCreated, statusJSON(status))
}

func (h *Handler) respond(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFrom(r.Context())
	if !ok {
		httpapi.WriteError(w, httpapi.Unauthorized("Требуется вход"))
		return
	}
	statusID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httpapi.WriteError(w, httpapi.BadRequest("invalid_id", "Некорректный ID"))
		return
	}
	if err := h.service.Respond(r.Context(), user.ID, statusID); err != nil {
		httpapi.WriteError(w, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *Handler) close(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFrom(r.Context())
	if !ok {
		httpapi.WriteError(w, httpapi.Unauthorized("Требуется вход"))
		return
	}
	statusID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httpapi.WriteError(w, httpapi.BadRequest("invalid_id", "Некорректный ID"))
		return
	}
	if err := h.service.Close(r.Context(), user.ID, statusID); err != nil {
		httpapi.WriteError(w, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
