package split

import (
	"github.com/danii1kq/srunal/internal/core/auth"
	"github.com/danii1kq/srunal/internal/platform/httpapi"
	"net/http"
	"strconv"
)

func (h *Handler) registerReminders(mux *http.ServeMux) {
	mux.Handle("GET /api/v1/split/debts/{id}/reminder", h.auth.Middleware(http.HandlerFunc(h.reminder)))
	mux.Handle("POST /api/v1/split/debts/{id}/reminder", h.auth.Middleware(http.HandlerFunc(h.reminder)))
	mux.Handle("GET /api/v1/split/bot-access", h.auth.Middleware(http.HandlerFunc(h.botAccess)))
	mux.Handle("POST /api/v1/split/bot-access", h.auth.Middleware(http.HandlerFunc(h.botAccess)))
}
func (h *Handler) reminder(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFrom(r.Context())
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httpapi.WriteError(w, httpapi.BadRequest("invalid_id", "Некорректный долг"))
		return
	}
	var a *ReminderAgreement
	if r.Method == "GET" {
		a, err = h.service.Reminder(r.Context(), user.ID, id)
	} else {
		var body struct {
			Version      int    `json:"version"`
			Action       string `json:"action"`
			Date         string `json:"date"`
			IntervalDays int    `json:"interval_days"`
		}
		if err = httpapi.DecodeJSON(w, r, &body, 4096); err != nil {
			httpapi.WriteError(w, err)
			return
		}
		if body.Action == "propose" {
			a, err = h.service.ProposeReminder(r.Context(), user.ID, id, body.Version, body.Date, body.IntervalDays)
		} else {
			a, err = h.service.RespondReminder(r.Context(), user.ID, id, body.Version, body.Action)
		}
	}
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}
	httpapi.WriteJSON(w, 200, map[string]any{"agreement": a})
}
func (h *Handler) botAccess(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFrom(r.Context())
	if r.Method == "POST" {
		var body struct {
			OptIn       *bool `json:"opt_in"`
			Enabled     *bool `json:"enabled"`
			ShowAmounts *bool `json:"show_amounts"`
		}
		if err := httpapi.DecodeJSON(w, r, &body, 1024); err != nil {
			httpapi.WriteError(w, err)
			return
		}
		if _, err := h.pool.Exec(r.Context(), `UPDATE user_settings SET bot_notifications_opt_in=coalesce($2,bot_notifications_opt_in),split_debt_notifications=coalesce($3,split_debt_notifications),split_debt_show_amounts=coalesce($4,split_debt_show_amounts) WHERE user_id=$1`, user.ID, body.OptIn, body.Enabled, body.ShowAmounts); err != nil {
			httpapi.WriteError(w, err)
			return
		}
	}
	var optIn, enabled, amounts bool
	if err := h.pool.QueryRow(r.Context(), `SELECT bot_notifications_opt_in,split_debt_notifications,split_debt_show_amounts FROM user_settings WHERE user_id=$1`, user.ID).Scan(&optIn, &enabled, &amounts); err != nil {
		httpapi.WriteError(w, err)
		return
	}
	httpapi.WriteJSON(w, 200, map[string]any{"opt_in": optIn, "enabled": enabled, "show_amounts": amounts, "bot_url": h.service.MiniAppURL})
}
