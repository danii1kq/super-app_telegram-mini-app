// Package user owns the profile and the shell settings of a SruNal account.
package user

import (
	"context"
	"net/http"

	"github.com/jackc/pgx/v5"

	"github.com/danii1kq/srunal/internal/core/auth"
	"github.com/danii1kq/srunal/internal/platform/httpapi"
	"github.com/danii1kq/srunal/internal/platform/postgres"
)

type Service struct {
	pool *postgres.Pool
}

func NewService(pool *postgres.Pool) *Service {
	return &Service{pool: pool}
}

type Profile struct {
	ID         int64
	PublicName string
	Username   string
	Lang       string
	Theme      string
	Timezone   string
	Pinned     []string
}

func (s *Service) Get(ctx context.Context, userID int64) (Profile, error) {
	var profile Profile
	err := s.pool.QueryRow(ctx,
		`SELECT u.id, u.public_name, coalesce(u.username, ''),
		        s.lang, s.theme, s.timezone, s.pinned_apps
		   FROM users u
		   LEFT JOIN user_settings s ON s.user_id = u.id
		  WHERE u.id = $1`, userID,
	).Scan(&profile.ID, &profile.PublicName, &profile.Username,
		&profile.Lang, &profile.Theme, &profile.Timezone, &profile.Pinned)
	if err == pgx.ErrNoRows {
		return Profile{}, httpapi.NotFound("Профиль не найден")
	}
	if err != nil {
		return Profile{}, err
	}
	return profile, nil
}

type SettingsPatch struct {
	Lang     *string   `json:"lang"`
	Theme    *string   `json:"theme"`
	Timezone *string   `json:"timezone"`
	Pinned   *[]string `json:"pinned_apps"`
}

func (s *Service) UpdateSettings(ctx context.Context, userID int64, patch SettingsPatch) error {
	fields := map[string]string{}
	if patch.Lang != nil {
		if *patch.Lang != "ru" && *patch.Lang != "kk" && *patch.Lang != "en" {
			return httpapi.Validation(map[string]string{"lang": "Допустимо: ru, kk, en"}, "Проверьте поля")
		}
		fields["lang"] = *patch.Lang
	}
	if patch.Theme != nil {
		if *patch.Theme != "auto" && *patch.Theme != "light" && *patch.Theme != "dark" {
			return httpapi.Validation(map[string]string{"theme": "Допустимо: auto, light, dark"}, "Проверьте поля")
		}
		fields["theme"] = *patch.Theme
	}

	_, err := s.pool.Exec(ctx,
		`UPDATE user_settings SET
		     lang     = coalesce($2, lang),
		     theme    = coalesce($3, theme),
		     timezone = coalesce($4, timezone),
		     pinned_apps = coalesce($5, pinned_apps),
		     updated_at = now()
		   WHERE user_id = $1`,
		userID, patch.Lang, patch.Theme, patch.Timezone, patch.Pinned,
	)
	return err
}

// Handler exposes /me.
type Handler struct {
	service *Service
	auth    *auth.Service
}

func NewHandler(service *Service, authService *auth.Service) *Handler {
	return &Handler{service: service, auth: authService}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.Handle("GET /api/v1/me", h.auth.Middleware(http.HandlerFunc(h.getMe)))
	mux.Handle("PATCH /api/v1/me/settings", h.auth.Middleware(http.HandlerFunc(h.patchSettings)))
}

func (h *Handler) getMe(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFrom(r.Context())
	if !ok {
		httpapi.WriteError(w, httpapi.Unauthorized("Требуется вход"))
		return
	}
	profile, err := h.service.Get(r.Context(), user.ID)
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, map[string]any{
		"id":          formatID(profile.ID),
		"public_name": profile.PublicName,
		"username":    profile.Username,
		"lang":        profile.Lang,
		"theme":       profile.Theme,
		"timezone":    profile.Timezone,
		"pinned_apps": profile.Pinned,
	})
}

func (h *Handler) patchSettings(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFrom(r.Context())
	if !ok {
		httpapi.WriteError(w, httpapi.Unauthorized("Требуется вход"))
		return
	}
	var patch SettingsPatch
	if err := httpapi.DecodeJSON(w, r, &patch, 16<<10); err != nil {
		httpapi.WriteError(w, err)
		return
	}
	if err := h.service.UpdateSettings(r.Context(), user.ID, patch); err != nil {
		httpapi.WriteError(w, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func formatID(id int64) string {
	if id == 0 {
		return "0"
	}
	const digits = "0123456789"
	var buffer [20]byte
	position := len(buffer)
	for id > 0 {
		position--
		buffer[position] = digits[id%10]
		id /= 10
	}
	return string(buffer[position:])
}
