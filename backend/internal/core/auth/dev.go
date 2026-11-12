package auth

import (
	"net/http"

	"github.com/danii1kq/srunal/internal/platform/httpapi"
)

// DevHandler is the local-only login used when the app runs outside Telegram.
//
// It exists so the Mini App can be opened in a browser during development. It is
// mounted only when SRUNAL_DEV_INSECURE_KEYS=1, and the service refuses to start
// in production without real keys, so this route cannot appear there.
type DevHandler struct {
	service *Service
}

func NewDevHandler(service *Service) *DevHandler {
	return &DevHandler{service: service}
}

func (h *DevHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/auth/dev", h.login)
}

type devLoginRequest struct {
	// Name lets the developer switch between two accounts to test a dialog.
	Name string `json:"name"`
}

func (h *DevHandler) login(w http.ResponseWriter, r *http.Request) {
	var body devLoginRequest
	if err := httpapi.DecodeJSON(w, r, &body, 4<<10); err != nil {
		httpapi.WriteError(w, err)
		return
	}
	if body.Name == "" {
		body.Name = "Разработчик"
	}

	// A stable negative id per name: real Telegram ids are positive, so a dev
	// account can never collide with a real one.
	var telegramID int64 = -1
	for _, char := range body.Name {
		telegramID = telegramID*31 - int64(char)
	}

	user, token, err := h.service.Login(r.Context(), InitData{
		TelegramID: telegramID,
		FirstName:  body.Name,
		Raw:        "dev",
	})
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
