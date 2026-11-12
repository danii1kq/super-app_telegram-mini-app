package day

import (
	"net/http"
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
	mux.Handle("GET /api/v1/apps/day/status", h.auth.Middleware(http.HandlerFunc(h.status)))
	mux.Handle("GET /api/v1/apps/day/events", h.auth.Middleware(http.HandlerFunc(h.events)))
}

func (h *Handler) status(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFrom(r.Context())
	if !ok {
		httpapi.WriteError(w, httpapi.Unauthorized("Требуется вход"))
		return
	}
	linked, err := h.service.Linked(r.Context(), user.ID)
	if err != nil {
		httpapi.WriteError(w, httpapi.Internal(err))
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, map[string]bool{"linked": linked})
}

func (h *Handler) events(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFrom(r.Context())
	if !ok {
		httpapi.WriteError(w, httpapi.Unauthorized("Требуется вход"))
		return
	}

	from := time.Now()
	if raw := r.URL.Query().Get("from"); raw != "" {
		parsed, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			httpapi.WriteError(w, httpapi.BadRequest("invalid_from", "Некорректная дата"))
			return
		}
		from = parsed
	}

	initData, err := h.auth.InitDataFor(r.Context(), user.ID, auth.BearerToken(r))
	if err != nil {
		httpapi.WriteError(w, httpapi.Internal(err))
		return
	}

	events, err := h.service.Week(r.Context(), user.ID, initData, from)
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, map[string]any{"events": events})
}
