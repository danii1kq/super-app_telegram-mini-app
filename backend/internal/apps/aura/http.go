package aura

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
	mux.Handle("GET /api/v1/aura", h.auth.Middleware(http.HandlerFunc(h.state)))
	mux.Handle("POST /api/v1/aura/farm", h.auth.Middleware(http.HandlerFunc(h.farm)))
	mux.Handle("POST /api/v1/aura/start", h.auth.Middleware(http.HandlerFunc(h.start)))
	mux.Handle("POST /api/v1/aura/answer", h.auth.Middleware(http.HandlerFunc(h.answer)))
	mux.Handle("POST /api/v1/aura/class", h.auth.Middleware(http.HandlerFunc(h.chooseClass)))
	mux.Handle("POST /api/v1/aura/relics/equip", h.auth.Middleware(http.HandlerFunc(h.equip)))
	mux.Handle("POST /api/v1/aura/stats", h.auth.Middleware(http.HandlerFunc(h.spendStat)))
	mux.Handle("GET /api/v1/aura/top", h.auth.Middleware(http.HandlerFunc(h.top)))
}

// state returns the hero plus the static catalogs, so the client needs one call
// to render the whole screen.
func (h *Handler) state(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFrom(r.Context())
	if !ok {
		httpapi.WriteError(w, httpapi.Unauthorized("Требуется вход"))
		return
	}
	player, err := h.service.snapshot(r.Context(), user.ID)
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}
	relics, err := h.service.relics(r.Context(), user.ID)
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, map[string]any{
		"player":  player,
		"relics":  relics,
		"classes": classCatalog,
		"catalog": relicCatalog,
	})
}

func (h *Handler) farm(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFrom(r.Context())
	if !ok {
		httpapi.WriteError(w, httpapi.Unauthorized("Требуется вход"))
		return
	}
	result, err := h.service.Farm(r.Context(), user.ID)
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, result)
}

type startRequest struct {
	Kind string `json:"kind"`
}

func (h *Handler) start(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFrom(r.Context())
	if !ok {
		httpapi.WriteError(w, httpapi.Unauthorized("Требуется вход"))
		return
	}
	var body startRequest
	if err := httpapi.DecodeJSON(w, r, &body, 4<<10); err != nil {
		httpapi.WriteError(w, err)
		return
	}
	result, err := h.service.Start(r.Context(), user.ID, SessionKind(body.Kind))
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, result)
}

type answerRequest struct {
	Choice    int    `json:"choice"`
	RequestID string `json:"request_id"`
}

func (h *Handler) answer(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFrom(r.Context())
	if !ok {
		httpapi.WriteError(w, httpapi.Unauthorized("Требуется вход"))
		return
	}
	var body answerRequest
	if err := httpapi.DecodeJSON(w, r, &body, 4<<10); err != nil {
		httpapi.WriteError(w, err)
		return
	}
	result, err := h.service.Answer(r.Context(), user.ID, body.Choice, body.RequestID)
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, result)
}

type classRequest struct {
	Class string `json:"class"`
}

func (h *Handler) chooseClass(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFrom(r.Context())
	if !ok {
		httpapi.WriteError(w, httpapi.Unauthorized("Требуется вход"))
		return
	}
	var body classRequest
	if err := httpapi.DecodeJSON(w, r, &body, 4<<10); err != nil {
		httpapi.WriteError(w, err)
		return
	}
	player, err := h.service.ChooseClass(r.Context(), user.ID, HeroClass(body.Class))
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, map[string]any{"player": player})
}

type equipRequest struct {
	Key  string `json:"key"`
	Slot string `json:"slot"`
}

func (h *Handler) equip(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFrom(r.Context())
	if !ok {
		httpapi.WriteError(w, httpapi.Unauthorized("Требуется вход"))
		return
	}
	var body equipRequest
	if err := httpapi.DecodeJSON(w, r, &body, 4<<10); err != nil {
		httpapi.WriteError(w, err)
		return
	}
	if err := h.service.Equip(r.Context(), user.ID, body.Key, Slot(body.Slot)); err != nil {
		httpapi.WriteError(w, err)
		return
	}
	relics, err := h.service.relics(r.Context(), user.ID)
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, map[string]any{"relics": relics})
}

type statRequest struct {
	Stat string `json:"stat"`
}

func (h *Handler) spendStat(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFrom(r.Context())
	if !ok {
		httpapi.WriteError(w, httpapi.Unauthorized("Требуется вход"))
		return
	}
	var body statRequest
	if err := httpapi.DecodeJSON(w, r, &body, 4<<10); err != nil {
		httpapi.WriteError(w, err)
		return
	}
	player, err := h.service.SpendStat(r.Context(), user.ID, body.Stat)
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, map[string]any{"player": player})
}

func (h *Handler) top(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFrom(r.Context())
	if !ok {
		httpapi.WriteError(w, httpapi.Unauthorized("Требуется вход"))
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	entries, err := h.service.Top(r.Context(), user.ID, limit)
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, map[string]any{"entries": entries})
}
