package tinder

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
	mux.Handle("GET /api/v1/tinder/profile", h.auth.Middleware(http.HandlerFunc(h.ownProfile)))
	mux.Handle("PUT /api/v1/tinder/profile", h.auth.Middleware(http.HandlerFunc(h.saveProfile)))
	mux.Handle("GET /api/v1/tinder/deck", h.auth.Middleware(http.HandlerFunc(h.deck)))
	mux.Handle("POST /api/v1/tinder/likes", h.auth.Middleware(http.HandlerFunc(h.like)))
	mux.Handle("POST /api/v1/tinder/passes", h.auth.Middleware(http.HandlerFunc(h.pass)))
	mux.Handle("GET /api/v1/tinder/matches", h.auth.Middleware(http.HandlerFunc(h.matches)))
}

func cardJSON(card Card) map[string]any {
	return map[string]any{
		"user_id":    strconv.FormatInt(card.UserID, 10),
		"name":       card.Name,
		"username":   card.Username,
		"bio":        card.Bio,
		"photo_url":  card.PhotoURL,
		"university": card.University,
		"age":        card.Age,
		"likes_you":  card.LikesYou,
		"created_at": card.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func (h *Handler) ownProfile(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFrom(r.Context())
	if !ok {
		httpapi.WriteError(w, httpapi.Unauthorized("Требуется вход"))
		return
	}
	profile, exists, err := h.service.OwnProfile(r.Context(), user.ID)
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}
	likes, err := h.service.LikesMe(r.Context(), user.ID)
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}
	if !exists {
		httpapi.WriteJSON(w, http.StatusOK, map[string]any{"profile": nil, "likes_me": likes})
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, map[string]any{
		"profile": map[string]any{
			"user_id":   strconv.FormatInt(profile.UserID, 10),
			"bio":       profile.Bio,
			"photo_url": profile.PhotoURL,
			"university": profile.University,
			"age":       profile.Age,
			"active":    profile.Active,
		},
		"likes_me": likes,
	})
}

type profileRequest struct {
	Bio      string `json:"bio"`
	PhotoURL string `json:"photo_url"`
	University string `json:"university"`
	Age      int    `json:"age"`
	Active   *bool  `json:"active"`
}

func (h *Handler) saveProfile(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFrom(r.Context())
	if !ok {
		httpapi.WriteError(w, httpapi.Unauthorized("Требуется вход"))
		return
	}
	var body profileRequest
	if err := httpapi.DecodeJSON(w, r, &body, 16<<10); err != nil {
		httpapi.WriteError(w, err)
		return
	}
	active := true
	if body.Active != nil {
		active = *body.Active
	}
	profile, err := h.service.SaveProfile(r.Context(), user.ID, body.Bio, body.PhotoURL, body.University, body.Age, active)
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, map[string]any{
		"profile": map[string]any{
			"user_id":   strconv.FormatInt(profile.UserID, 10),
			"bio":       profile.Bio,
			"photo_url": profile.PhotoURL,
			"university": profile.University,
			"age":       profile.Age,
			"active":    profile.Active,
		},
	})
}

func (h *Handler) deck(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFrom(r.Context())
	if !ok {
		httpapi.WriteError(w, httpapi.Unauthorized("Требуется вход"))
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	cards, err := h.service.Deck(r.Context(), user.ID, limit)
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}
	out := make([]map[string]any, 0, len(cards))
	for _, card := range cards {
		out = append(out, cardJSON(card))
	}
	httpapi.WriteJSON(w, http.StatusOK, map[string]any{"cards": out})
}

type targetRequest struct {
	UserID string `json:"user_id"`
}

func (h *Handler) like(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFrom(r.Context())
	if !ok {
		httpapi.WriteError(w, httpapi.Unauthorized("Требуется вход"))
		return
	}
	var body targetRequest
	if err := httpapi.DecodeJSON(w, r, &body, 4<<10); err != nil {
		httpapi.WriteError(w, err)
		return
	}
	target, err := strconv.ParseInt(body.UserID, 10, 64)
	if err != nil {
		httpapi.WriteError(w, httpapi.BadRequest("invalid_id", "Некорректный ID"))
		return
	}
	matched, matchID, err := h.service.Like(r.Context(), user.ID, target)
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}
	out := map[string]any{"matched": matched}
	if matched {
		out["match_id"] = strconv.FormatInt(matchID, 10)
	}
	httpapi.WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) pass(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFrom(r.Context())
	if !ok {
		httpapi.WriteError(w, httpapi.Unauthorized("Требуется вход"))
		return
	}
	var body targetRequest
	if err := httpapi.DecodeJSON(w, r, &body, 4<<10); err != nil {
		httpapi.WriteError(w, err)
		return
	}
	target, err := strconv.ParseInt(body.UserID, 10, 64)
	if err != nil {
		httpapi.WriteError(w, httpapi.BadRequest("invalid_id", "Некорректный ID"))
		return
	}
	if err := h.service.Pass(r.Context(), user.ID, target); err != nil {
		httpapi.WriteError(w, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *Handler) matches(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFrom(r.Context())
	if !ok {
		httpapi.WriteError(w, httpapi.Unauthorized("Требуется вход"))
		return
	}
	list, err := h.service.Matches(r.Context(), user.ID)
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}
	out := make([]map[string]any, 0, len(list))
	for _, item := range list {
		out = append(out, map[string]any{
			"id":         strconv.FormatInt(item.ID, 10),
			"partner_id": strconv.FormatInt(item.PartnerID, 10),
			"name":       item.Name,
			"username":   item.Username,
			"photo_url":  item.PhotoURL,
			"created_at": item.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
	httpapi.WriteJSON(w, http.StatusOK, map[string]any{"matches": out})
}
