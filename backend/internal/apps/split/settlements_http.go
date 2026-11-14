package split

import (
	"github.com/danii1kq/srunal/internal/core/auth"
	"github.com/danii1kq/srunal/internal/platform/httpapi"
	"net/http"
	"strconv"
)

func (h *Handler) registerSettlements(mux *http.ServeMux) {
	mux.Handle("GET /api/v1/split/groups/{id}/settlements", h.auth.Middleware(http.HandlerFunc(h.settlements)))
	mux.Handle("POST /api/v1/split/groups/{id}/settlements", h.auth.Middleware(http.HandlerFunc(h.settlements)))
	mux.Handle("POST /api/v1/split/settlements/{id}/respond", h.auth.Middleware(http.HandlerFunc(h.settlements)))
}
func (h *Handler) settlements(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFrom(r.Context())
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httpapi.WriteError(w, httpapi.BadRequest("invalid_id", "Некорректный расчёт"))
		return
	}
	if r.Method == "GET" {
		pairs, proposals, err := h.service.SettlementPlan(r.Context(), user.ID, id)
		if err != nil {
			httpapi.WriteError(w, err)
			return
		}
		httpapi.WriteJSON(w, 200, map[string]any{"pairs": pairs, "proposals": proposals})
		return
	}
	var body struct {
		OtherID         string `json:"other_id"`
		GroupVersion    int64  `json:"group_version"`
		ClientRequestID string `json:"client_request_id"`
		Action          string `json:"action"`
	}
	if err = httpapi.DecodeJSON(w, r, &body, 4096); err != nil {
		httpapi.WriteError(w, err)
		return
	}
	var proposal Settlement
	if body.Action != "" {
		proposal, err = h.service.RespondSettlement(r.Context(), user.ID, id, body.Action)
	} else {
		var other int64
		other, err = strconv.ParseInt(body.OtherID, 10, 64)
		if err == nil {
			proposal, err = h.service.ProposeSettlement(r.Context(), user.ID, id, other, body.GroupVersion, body.ClientRequestID)
		} else {
			err = httpapi.BadRequest("invalid_member", "Некорректный участник")
		}
	}
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}
	httpapi.WriteJSON(w, 200, proposal)
}
