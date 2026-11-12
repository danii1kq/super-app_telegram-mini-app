package social

import (
	"net/http"
	"strconv"

	"github.com/danii1kq/srunal/internal/core/auth"
	"github.com/danii1kq/srunal/internal/platform/httpapi"
)

type commentDTO struct {
	ID         string `json:"id"`
	PostID     string `json:"post_id"`
	AuthorID   string `json:"author_id"`
	AuthorName string `json:"author_name"`
	Body       string `json:"body"`
	CreatedAt  string `json:"created_at"`
}

func commentToDTO(comment Comment) commentDTO {
	return commentDTO{ID: itoa(comment.ID), PostID: itoa(comment.PostID), AuthorID: itoa(comment.AuthorID),
		AuthorName: comment.AuthorName, Body: comment.Body, CreatedAt: comment.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00")}
}

func (h *Handler) comments(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFrom(r.Context())
	if !ok {
		httpapi.WriteError(w, httpapi.Unauthorized("Требуется вход"))
		return
	}
	postID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || postID <= 0 {
		httpapi.WriteError(w, httpapi.BadRequest("invalid_id", "Некорректный ID"))
		return
	}
	before, err := cursorID(r.URL.Query().Get("before"))
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}
	comments, err := h.service.Comments(r.Context(), user.ID, postID, before)
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}
	var next *string
	if len(comments) > 20 {
		comments = comments[:20]
		value := itoa(comments[19].ID)
		next = &value
	}
	out := make([]commentDTO, 0, len(comments))
	for _, comment := range comments {
		out = append(out, commentToDTO(comment))
	}
	httpapi.WriteJSON(w, http.StatusOK, map[string]any{"comments": out, "next_before": next})
}

func (h *Handler) createComment(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFrom(r.Context())
	if !ok {
		httpapi.WriteError(w, httpapi.Unauthorized("Требуется вход"))
		return
	}
	postID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || postID <= 0 {
		httpapi.WriteError(w, httpapi.BadRequest("invalid_id", "Некорректный ID"))
		return
	}
	var body struct {
		Body string `json:"body"`
	}
	if err := httpapi.DecodeJSON(w, r, &body, 16<<10); err != nil {
		httpapi.WriteError(w, err)
		return
	}
	comment, err := h.service.CreateComment(r.Context(), user.ID, postID, body.Body)
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusCreated, commentToDTO(comment))
}

func cursorID(raw string) (int64, error) {
	if raw == "" {
		return 0, nil
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value <= 0 {
		return 0, httpapi.BadRequest("invalid_cursor", "Некорректный курсор")
	}
	return value, nil
}
