package day

import (
	"net/http"
	"time"

	"github.com/danii1kq/srunal/internal/core/auth"
	"github.com/danii1kq/srunal/internal/platform/httpapi"
	"github.com/danii1kq/srunal/internal/platform/vision"
)

type Handler struct {
	service     *Service
	auth        *auth.Service
	weather     *WeatherClient
	vision      *vision.Client
	importSlots chan struct{}
}

func NewHandler(service *Service, authService *auth.Service, imageReader *vision.Client, weather *WeatherClient) *Handler {
	return &Handler{service: service, auth: authService, weather: weather, vision: imageReader, importSlots: make(chan struct{}, 2)}
}

func (h *Handler) Register(mux *http.ServeMux) {
	h.registerWeather(mux)
	mux.Handle("POST /api/v1/day/import/preview", h.auth.Middleware(http.HandlerFunc(h.importTimetable)))
	mux.Handle("GET /api/v1/apps/day/status", h.auth.Middleware(http.HandlerFunc(h.status)))
	mux.Handle("GET /api/v1/apps/day/events", h.auth.Middleware(http.HandlerFunc(h.events)))
	mux.Handle("POST /api/v1/apps/day/connect", h.auth.Middleware(http.HandlerFunc(h.connect)))
	mux.Handle("DELETE /api/v1/apps/day/connect", h.auth.Middleware(http.HandlerFunc(h.disconnect)))
	mux.Handle("POST /api/v1/day/events", h.auth.Middleware(http.HandlerFunc(h.create)))
	mux.Handle("GET /api/v1/day/events/{id}", h.auth.Middleware(http.HandlerFunc(h.event)))
	mux.Handle("PATCH /api/v1/day/events/{id}", h.auth.Middleware(http.HandlerFunc(h.update)))
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
	httpapi.WriteJSON(w, http.StatusOK, map[string]bool{"linked": linked, "configured": h.service.client.Configured(), "import_available": h.vision.Configured()})
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
	to := from.AddDate(0, 0, 7)
	if raw := r.URL.Query().Get("to"); raw != "" {
		parsed, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			httpapi.WriteError(w, httpapi.BadRequest("invalid_to", "Некорректная дата"))
			return
		}
		to = parsed
	}

	initData, err := h.auth.InitDataFor(r.Context(), user.ID, auth.BearerToken(r))
	if err != nil {
		httpapi.WriteError(w, httpapi.Internal(err))
		return
	}

	events, err := h.service.Window(r.Context(), user.ID, initData, from, to)
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, map[string]any{"events": events})
}

func (h *Handler) connect(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFrom(r.Context())
	if !ok {
		httpapi.WriteError(w, httpapi.Unauthorized("Требуется вход"))
		return
	}
	initData, err := h.auth.InitDataFor(r.Context(), user.ID, auth.BearerToken(r))
	if err == nil {
		err = h.service.Connect(r.Context(), user.ID, initData)
	}
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, map[string]bool{"linked": true})
}

func (h *Handler) disconnect(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFrom(r.Context())
	if !ok {
		httpapi.WriteError(w, httpapi.Unauthorized("Требуется вход"))
		return
	}
	initData, err := h.auth.InitDataFor(r.Context(), user.ID, auth.BearerToken(r))
	if err == nil {
		err = h.service.Disconnect(r.Context(), user.ID, initData)
	}
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, map[string]bool{"linked": false})
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFrom(r.Context())
	if !ok {
		httpapi.WriteError(w, httpapi.Unauthorized("Требуется вход"))
		return
	}
	var body struct {
		Title    string `json:"title"`
		StartsAt string `json:"starts_at"`
		EndsAt   string `json:"ends_at"`
		AllDay   bool   `json:"all_day"`
		Location string `json:"location"`
	}
	if err := httpapi.DecodeJSON(w, r, &body, 8<<10); err != nil {
		httpapi.WriteError(w, err)
		return
	}
	initData, err := h.auth.InitDataFor(r.Context(), user.ID, auth.BearerToken(r))
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}
	event, err := h.service.Create(r.Context(), user.ID, initData, CreateEventInput{Title: body.Title, StartsAt: body.StartsAt, EndsAt: body.EndsAt, AllDay: body.AllDay, Location: body.Location})
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusCreated, event)
}
