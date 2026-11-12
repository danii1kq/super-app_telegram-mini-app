package split

import (
	"net/http"
	"strconv"
	"time"

	"github.com/danii1kq/srunal/internal/core/auth"
	"github.com/danii1kq/srunal/internal/platform/httpapi"
)

// RegisterReport mounts the monthly report and the photo retention routes.
func (h *Handler) RegisterReport(mux *http.ServeMux) {
	mux.Handle("GET /api/v1/split/report", h.auth.Middleware(http.HandlerFunc(h.report)))
}

// report returns one month of the viewer's own spending.
//
// The month boundaries are computed in the profile's time zone, so the same
// request gives the same answer regardless of the server's zone.
func (h *Handler) report(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFrom(r.Context())
	if !ok {
		httpapi.WriteError(w, httpapi.Unauthorized("Требуется вход"))
		return
	}

	zone := r.URL.Query().Get("tz")
	if zone == "" {
		// The profile's zone is the default: the month must follow the person,
		// not the server.
		if err := h.pool.QueryRow(r.Context(),
			`SELECT timezone FROM user_settings WHERE user_id = $1`, user.ID,
		).Scan(&zone); err != nil {
			zone = "Asia/Almaty"
		}
	}

	report, err := h.service.MonthReport(r.Context(), user.ID, r.URL.Query().Get("month"), zone)
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, reportJSON(report))
}

func reportJSON(report Report) map[string]any {
	lines := make([]map[string]any, 0, len(report.Lines))
	for _, line := range report.Lines {
		lines = append(lines, map[string]any{
			"expense_id":  strconv.FormatInt(line.ExpenseID, 10),
			"group_id":    strconv.FormatInt(line.GroupID, 10),
			"group_title": line.GroupTitle,
			"description": line.Description,
			"spent_at":    line.SpentAt.UTC().Format(time.RFC3339),
			"my_share":    strconv.FormatInt(line.MyShare, 10),
			"paid_by_me":  strconv.FormatInt(line.PaidByMe, 10),
			"category":    line.Category,
		})
	}

	categories := make(map[string]string, len(report.ByCategory))
	for name, total := range report.ByCategory {
		categories[name] = strconv.FormatInt(total, 10)
	}

	return map[string]any{
		"from":           report.From.UTC().Format(time.RFC3339),
		"to":             report.To.UTC().Format(time.RFC3339),
		"my_spending":    strconv.FormatInt(report.MySpending, 10),
		"paid_by_me":     strconv.FormatInt(report.PaidByMe, 10),
		"by_category":    categories,
		"lines":          lines,
		"lent_out":       strconv.FormatInt(report.LentOut, 10),
		"returned_to_me": strconv.FormatInt(report.ReturnedToMe, 10),
		"returned_by_me": strconv.FormatInt(report.ReturnedByMe, 10),
		"owed_to_me_now": strconv.FormatInt(report.OwedToMeNow, 10),
		"i_owe_now":      strconv.FormatInt(report.IOweNow, 10),
	}
}
