package social

import (
	"net/http"
	"strconv"

	"github.com/danii1kq/srunal/internal/core/auth"
	"github.com/danii1kq/srunal/internal/platform/httpapi"
)

type Handler struct {
	service *Service
	auth    *auth.Service
}

func NewHandler(service *Service, authService *auth.Service) *Handler {
	return &Handler{service: service, auth: authService}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.Handle("GET /api/v1/feed", h.auth.Middleware(http.HandlerFunc(h.feed)))
	mux.Handle("POST /api/v1/posts", h.auth.Middleware(http.HandlerFunc(h.createPost)))
	mux.Handle("GET /api/v1/posts/{id}", h.auth.Middleware(http.HandlerFunc(h.post)))
	mux.Handle("POST /api/v1/posts/{id}/view", h.auth.Middleware(http.HandlerFunc(h.view)))
	mux.Handle("GET /api/v1/posts/{id}/comments", h.auth.Middleware(http.HandlerFunc(h.comments)))
	mux.Handle("POST /api/v1/posts/{id}/comments", h.auth.Middleware(http.HandlerFunc(h.createComment)))
	mux.Handle("DELETE /api/v1/posts/{id}", h.auth.Middleware(http.HandlerFunc(h.deletePost)))
	mux.Handle("POST /api/v1/posts/{id}/reactions", h.auth.Middleware(http.HandlerFunc(h.react)))
	mux.Handle("POST /api/v1/authors/{id}/follow", h.auth.Middleware(http.HandlerFunc(h.follow)))
	mux.Handle("DELETE /api/v1/authors/{id}/follow", h.auth.Middleware(http.HandlerFunc(h.unfollow)))
	mux.Handle("GET /api/v1/people", h.auth.Middleware(http.HandlerFunc(h.people)))
	mux.Handle("GET /api/v1/people/{id}", h.auth.Middleware(http.HandlerFunc(h.person)))
	mux.Handle("GET /api/v1/people/{id}/posts", h.auth.Middleware(http.HandlerFunc(h.authorPosts)))
}

type postDTO struct {
	ID          string `json:"id"`
	AuthorID    string `json:"author_id"`
	AuthorName  string `json:"author_name"`
	Body        string `json:"body"`
	Audience    string `json:"audience"`
	CommunityID string `json:"community_id,omitempty"`
	CreatedAt   string `json:"created_at"`
	Likes       int    `json:"likes"`
	Comments    int    `json:"comments"`
	LikedByMe   bool   `json:"liked_by_me"`
	Views       int    `json:"views"`
}

func toDTO(post Post) postDTO {
	dto := postDTO{
		ID:         strconv.FormatInt(post.ID, 10),
		AuthorID:   strconv.FormatInt(post.AuthorID, 10),
		AuthorName: post.AuthorName,
		Body:       post.Body,
		Audience:   post.Audience,
		CreatedAt:  post.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
		Likes:      post.Likes,
		Comments:   post.Comments,
		LikedByMe:  post.LikedByMe,
		Views:      post.Views,
	}
	if post.CommunityID != nil {
		dto.CommunityID = strconv.FormatInt(*post.CommunityID, 10)
	}
	return dto
}

func (h *Handler) feed(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFrom(r.Context())
	if !ok {
		httpapi.WriteError(w, httpapi.Unauthorized("Требуется вход"))
		return
	}

	before, err := cursorID(r.URL.Query().Get("before"))
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 50 {
		limit = 20
	}

	posts, err := h.service.Feed(r.Context(), user.ID, before, limit+1)
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}

	var next *string
	if len(posts) > limit {
		posts = posts[:limit]
		value := itoa(posts[limit-1].ID)
		next = &value
	}
	out := make([]postDTO, 0, len(posts))
	for _, post := range posts {
		out = append(out, toDTO(post))
	}
	httpapi.WriteJSON(w, http.StatusOK, map[string]any{"posts": out, "next_before": next})
}

type createPostRequest struct {
	Body      string `json:"body"`
	Audience  string `json:"audience"`
	Community string `json:"community_id"`
}

func (h *Handler) createPost(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFrom(r.Context())
	if !ok {
		httpapi.WriteError(w, httpapi.Unauthorized("Требуется вход"))
		return
	}

	var body createPostRequest
	if err := httpapi.DecodeJSON(w, r, &body, 32<<10); err != nil {
		httpapi.WriteError(w, err)
		return
	}

	input := NewPost{Body: body.Body, Audience: body.Audience}
	if body.Community != "" {
		id, err := strconv.ParseInt(body.Community, 10, 64)
		if err != nil {
			httpapi.WriteError(w, httpapi.BadRequest("invalid_community", "Некорректный ID сообщества"))
			return
		}
		input.Community = &id
	}

	post, err := h.service.CreatePost(r.Context(), user.ID, input)
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusCreated, toDTO(post))
}

func (h *Handler) deletePost(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFrom(r.Context())
	if !ok {
		httpapi.WriteError(w, httpapi.Unauthorized("Требуется вход"))
		return
	}
	postID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httpapi.WriteError(w, httpapi.BadRequest("invalid_id", "Некорректный ID"))
		return
	}
	if err := h.service.DeletePost(r.Context(), user.ID, postID); err != nil {
		httpapi.WriteError(w, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

type reactionRequest struct {
	Kind string `json:"kind"`
}

func (h *Handler) react(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFrom(r.Context())
	if !ok {
		httpapi.WriteError(w, httpapi.Unauthorized("Требуется вход"))
		return
	}
	postID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httpapi.WriteError(w, httpapi.BadRequest("invalid_id", "Некорректный ID"))
		return
	}
	var body reactionRequest
	if err := httpapi.DecodeJSON(w, r, &body, 4<<10); err != nil {
		httpapi.WriteError(w, err)
		return
	}
	added, err := h.service.ToggleReaction(r.Context(), user.ID, postID, body.Kind)
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, map[string]bool{"added": added})
}

func (h *Handler) follow(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFrom(r.Context())
	if !ok {
		httpapi.WriteError(w, httpapi.Unauthorized("Требуется вход"))
		return
	}
	authorID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httpapi.WriteError(w, httpapi.BadRequest("invalid_id", "Некорректный ID"))
		return
	}
	if err := h.service.Follow(r.Context(), user.ID, authorID); err != nil {
		httpapi.WriteError(w, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *Handler) unfollow(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFrom(r.Context())
	if !ok {
		httpapi.WriteError(w, httpapi.Unauthorized("Требуется вход"))
		return
	}
	authorID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httpapi.WriteError(w, httpapi.BadRequest("invalid_id", "Некорректный ID"))
		return
	}
	if err := h.service.Unfollow(r.Context(), user.ID, authorID); err != nil {
		httpapi.WriteError(w, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
