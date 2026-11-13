package aura

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
	mux.Handle("GET /api/v1/aura", h.auth.Middleware(http.HandlerFunc(h.state)))
	mux.Handle("POST /api/v1/aura/farm", h.auth.Middleware(http.HandlerFunc(h.farm)))
	mux.Handle("POST /api/v1/aura/start", h.auth.Middleware(http.HandlerFunc(h.start)))
	mux.Handle("POST /api/v1/aura/answer", h.auth.Middleware(http.HandlerFunc(h.answer)))
	mux.Handle("POST /api/v1/aura/class", h.auth.Middleware(http.HandlerFunc(h.chooseClass)))
	mux.Handle("POST /api/v1/aura/relics/equip", h.auth.Middleware(http.HandlerFunc(h.equip)))
	mux.Handle("POST /api/v1/aura/stats", h.auth.Middleware(http.HandlerFunc(h.spendStat)))
	mux.Handle("POST /api/v1/aura/route", h.auth.Middleware(http.HandlerFunc(h.extraAction)))
	mux.Handle("POST /api/v1/aura/ability", h.auth.Middleware(http.HandlerFunc(h.extraAction)))
	mux.Handle("POST /api/v1/aura/difficulty", h.auth.Middleware(http.HandlerFunc(h.extraAction)))
	mux.Handle("POST /api/v1/aura/relics/upgrade", h.auth.Middleware(http.HandlerFunc(h.extraAction)))
	mux.Handle("POST /api/v1/aura/relics/craft", h.auth.Middleware(http.HandlerFunc(h.extraAction)))
	mux.Handle("POST /api/v1/aura/quests/claim", h.auth.Middleware(http.HandlerFunc(h.extraAction)))
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
	data, err := h.service.load(r.Context(), user.ID)
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, map[string]any{
		"player":      player,
		"relics":      relics,
		"classes":     classCatalog,
		"catalog":     relicsCatalog(),
		"dungeons":    dungeons,
		"quests":      questsFor(data),
		"server_time": time.Now().UTC(),
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
	Dungeon    string `json:"dungeon"`
	Difficulty string `json:"difficulty"`
	Kind       string `json:"kind"`
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
	result, err := h.service.StartDungeon(r.Context(), user.ID, SessionKind(body.Kind), body.Dungeon, body.Difficulty)
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, result)
}

type answerRequest struct {
	QuestionID string `json:"question_id"`
	Choice     int    `json:"choice"`
	RequestID  string `json:"request_id"`
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
	result, err := h.service.Answer(r.Context(), user.ID, body.Choice, body.RequestID, body.QuestionID)
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
	httpapi.WriteJSON(w, http.StatusOK, map[string]any{"entries": entries, "ends_at": auraWeekStart(h.service.now()).AddDate(0, 0, 7)})
}

func (h *Handler) extraAction(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFrom(r.Context())
	var body struct {
		Route      string `json:"route"`
		RequestID  string `json:"request_id"`
		Difficulty string `json:"difficulty"`
		Key        string `json:"key"`
		ID         string `json:"id"`
	}
	if err := httpapi.DecodeJSON(w, r, &body, 4096); err != nil {
		httpapi.WriteError(w, err)
		return
	}
	var result Result
	var err error
	switch r.URL.Path {
	case "/api/v1/aura/route":
		result, err = h.service.Route(r.Context(), user.ID, body.Route, body.RequestID)
	case "/api/v1/aura/ability":
		result, err = h.service.Ability(r.Context(), user.ID, body.RequestID)
	case "/api/v1/aura/difficulty":
		result, err = h.service.Difficulty(r.Context(), user.ID, body.Difficulty)
	case "/api/v1/aura/relics/upgrade":
		result, err = h.service.RelicUpgrade(r.Context(), user.ID, body.Key)
	case "/api/v1/aura/relics/craft":
		result, err = h.service.RelicCraft(r.Context(), user.ID, body.Key)
	case "/api/v1/aura/quests/claim":
		result, err = h.service.QuestClaim(r.Context(), user.ID, body.ID)
	}
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, result)
}
