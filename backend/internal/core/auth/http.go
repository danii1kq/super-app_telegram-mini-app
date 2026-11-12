package auth

import (
	"net/http"

	"github.com/danii1kq/srunal/internal/platform/httpapi"
)

// Handler exposes the auth endpoints.
type Handler struct {
	verifier *Verifier
	service  *Service
}

func NewHandler(verifier *Verifier, service *Service) *Handler {
	return &Handler{verifier: verifier, service: service}
}

// Register mounts the public auth routes on mux.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/auth/telegram", h.login)
	mux.HandleFunc("POST /api/v1/auth/logout", h.logout)
}

type loginRequest struct {
	InitData string `json:"init_data"`
}

type loginResponse struct {
	Token string `json:"token"`
	Me    meDTO  `json:"me"`
}

type meDTO struct {
	ID         string `json:"id"`
	PublicName string `json:"public_name"`
	Username   string `json:"username,omitempty"`
	Lang       string `json:"lang"`
	Theme      string `json:"theme"`
	Timezone   string `json:"timezone"`
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var body loginRequest
	if err := httpapi.DecodeJSON(w, r, &body, 64<<10); err != nil {
		httpapi.WriteError(w, err)
		return
	}
	if body.InitData == "" {
		httpapi.WriteError(w, httpapi.BadRequest("missing_init_data", "Не передан initData"))
		return
	}

	data, err := h.verifier.Verify(body.InitData)
	if err != nil {
		httpapi.WriteError(w, httpapi.Unauthorized("Не удалось подтвердить вход через Telegram"))
		return
	}

	user, token, err := h.service.Login(r.Context(), data)
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}

	settings, err := h.service.Settings(r.Context(), user.ID)
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}

	httpapi.WriteJSON(w, http.StatusOK, loginResponse{
		Token: token,
		Me: meDTO{
			ID:         formatID(user.ID),
			PublicName: user.PublicName,
			Username:   user.Username,
			Lang:       settings.Lang,
			Theme:      settings.Theme,
			Timezone:   settings.Timezone,
		},
	})
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	if err := h.service.Logout(r.Context(), BearerToken(r)); err != nil {
		httpapi.WriteError(w, httpapi.Internal(err))
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// formatID renders a bigint as a string: JavaScript numbers lose precision
// above 2^53.
func formatID(id int64) string {
	const digits = "0123456789"
	if id == 0 {
		return "0"
	}
	negative := id < 0
	if negative {
		id = -id
	}
	var buffer [20]byte
	position := len(buffer)
	for id > 0 {
		position--
		buffer[position] = digits[id%10]
		id /= 10
	}
	if negative {
		position--
		buffer[position] = '-'
	}
	return string(buffer[position:])
}
