package social

import (
	"net/http"
	"strconv"

	"github.com/danii1kq/srunal/internal/core/auth"
	"github.com/danii1kq/srunal/internal/platform/httpapi"
)

// people lists the directory: everyone the viewer can write to.
func (h *Handler) people(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFrom(r.Context())
	if !ok {
		httpapi.WriteError(w, httpapi.Unauthorized("Требуется вход"))
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	people, err := h.service.People(r.Context(), user.ID, r.URL.Query().Get("q"), limit)
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}
	out := make([]personDTO, 0, len(people))
	for _, person := range people {
		out = append(out, toPersonDTO(person))
	}
	httpapi.WriteJSON(w, http.StatusOK, map[string]any{"people": out})
}

func (h *Handler) person(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFrom(r.Context())
	if !ok {
		httpapi.WriteError(w, httpapi.Unauthorized("Требуется вход"))
		return
	}
	userID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httpapi.WriteError(w, httpapi.BadRequest("invalid_id", "Некорректный ID"))
		return
	}
	person, err := h.service.Profile(r.Context(), user.ID, userID)
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, toProfileDTO(person))
}
