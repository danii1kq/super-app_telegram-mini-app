package doc

import (
	"net/http"

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
	mux.Handle("GET /api/v1/apps/doc/orders", h.auth.Middleware(http.HandlerFunc(h.list)))
	mux.Handle("GET /api/v1/apps/doc/orders/{number}", h.auth.Middleware(http.HandlerFunc(h.one)))
	mux.Handle("GET /api/v1/apps/doc/status", h.auth.Middleware(http.HandlerFunc(h.status)))
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFrom(r.Context())
	if !ok {
		httpapi.WriteError(w, httpapi.Unauthorized("Требуется вход"))
		return
	}
	orders, err := h.service.Orders(r.Context(), user.ID)
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, map[string]any{"orders": orders})
}

func (h *Handler) one(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFrom(r.Context())
	if !ok {
		httpapi.WriteError(w, httpapi.Unauthorized("Требуется вход"))
		return
	}
	order, err := h.service.Order(r.Context(), user.ID, r.PathValue("number"))
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, order)
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
