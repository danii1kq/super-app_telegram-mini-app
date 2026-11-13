package split

import (
	"github.com/danii1kq/srunal/internal/core/auth"
	"github.com/danii1kq/srunal/internal/platform/httpapi"
	"net/http"
	"strconv"
)

func (h *Handler) respondPayment(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFrom(r.Context())
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httpapi.WriteError(w, httpapi.BadRequest("invalid_id", "Некорректный долг"))
		return
	}
	pid, err := strconv.ParseInt(r.PathValue("paymentId"), 10, 64)
	if err != nil {
		httpapi.WriteError(w, httpapi.BadRequest("invalid_id", "Некорректный перевод"))
		return
	}
	var body struct {
		Accept *bool `json:"accept"`
	}
	if err = httpapi.DecodeJSON(w, r, &body, 1024); err != nil {
		httpapi.WriteError(w, err)
		return
	}
	if body.Accept == nil {
		httpapi.WriteError(w, httpapi.BadRequest("invalid_decision", "Укажите ответ"))
		return
	}
	debt, err := h.service.RespondPayment(r.Context(), user.ID, id, pid, *body.Accept)
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}
	httpapi.WriteJSON(w, 200, debtJSON(debt))
}
