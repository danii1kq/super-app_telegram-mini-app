package social

import (
	"net/http"

	"github.com/danii1kq/srunal/internal/core/auth"
	"github.com/danii1kq/srunal/internal/platform/httpapi"
)

func (h *Handler) post(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFrom(r.Context())
	id, err := cursorID(r.PathValue("id"))
	if err != nil || id == 0 {
		httpapi.WriteError(w, httpapi.BadRequest("invalid_id", "Некорректный ID"))
		return
	}
	post, err := h.service.byID(r.Context(), id, user.ID)
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, toDTO(post))
}

func (h *Handler) view(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFrom(r.Context())
	id, err := cursorID(r.PathValue("id"))
	if err != nil || id == 0 {
		httpapi.WriteError(w, httpapi.BadRequest("invalid_id", "Некорректный ID"))
		return
	}
	count, err := h.service.RecordView(r.Context(), user.ID, id)
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, map[string]int{"views": count})
}

func (h *Handler) authorPosts(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFrom(r.Context())
	id, err := cursorID(r.PathValue("id"))
	if err != nil || id == 0 {
		httpapi.WriteError(w, httpapi.BadRequest("invalid_id", "Некорректный ID"))
		return
	}
	before, err := cursorID(r.URL.Query().Get("before"))
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}
	posts, err := h.service.AuthorPosts(r.Context(), user.ID, id, before, 21)
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}
	var next *string
	if len(posts) > 20 {
		posts = posts[:20]
		value := itoa(posts[19].ID)
		next = &value
	}
	out := make([]postDTO, 0, len(posts))
	for _, post := range posts {
		out = append(out, toDTO(post))
	}
	httpapi.WriteJSON(w, http.StatusOK, map[string]any{"posts": out, "next_before": next})
}
