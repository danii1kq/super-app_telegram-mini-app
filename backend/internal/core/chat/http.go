package chat

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"

	"github.com/danii1kq/srunal/internal/core/auth"
	"github.com/danii1kq/srunal/internal/platform/httpapi"
)

type Handler struct {
	service *Service
	auth    *auth.Service
	events  *Hub
}

func NewHandler(service *Service, authService *auth.Service, events *Hub) *Handler {
	return &Handler{service: service, auth: authService, events: events}
}

func (h *Handler) Register(mux *http.ServeMux) {
	h.registerAttachments(mux)
	h.registerGroups(mux)
	mux.Handle("GET /api/v1/conversations", h.auth.Middleware(http.HandlerFunc(h.list)))
	mux.Handle("POST /api/v1/conversations", h.auth.Middleware(http.HandlerFunc(h.openDirect)))
	mux.Handle("GET /api/v1/conversations/{id}/messages", h.auth.Middleware(http.HandlerFunc(h.history)))
	mux.Handle("POST /api/v1/conversations/{id}/messages", h.auth.Middleware(http.HandlerFunc(h.send)))
	mux.Handle("POST /api/v1/conversations/{id}/read", h.auth.Middleware(http.HandlerFunc(h.markRead)))
	mux.Handle("GET /api/v1/events/stream", h.auth.Middleware(http.HandlerFunc(h.stream)))
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFrom(r.Context())
	if !ok {
		httpapi.WriteError(w, httpapi.Unauthorized("Требуется вход"))
		return
	}
	conversations, err := h.service.List(r.Context(), user.ID)
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}
	out := make([]map[string]any, 0, len(conversations))
	for _, conversation := range conversations {
		out = append(out, conversationJSON(conversation))
	}
	httpapi.WriteJSON(w, http.StatusOK, map[string]any{"conversations": out})
}

func (h *Handler) history(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFrom(r.Context())
	if !ok {
		httpapi.WriteError(w, httpapi.Unauthorized("Требуется вход"))
		return
	}
	conversationID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || conversationID <= 0 {
		httpapi.WriteError(w, httpapi.BadRequest("invalid_id", "Некорректный ID"))
		return
	}
	query, err := parseHistoryQuery(r.URL.Query())
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}

	messages, hasMore, err := h.service.History(r.Context(), user.ID, conversationID, query)
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}
	out := make([]map[string]any, 0, len(messages))
	for _, message := range messages {
		out = append(out, messageJSON(message))
	}
	httpapi.WriteJSON(w, http.StatusOK, map[string]any{"messages": out, "has_more": hasMore})
}

type sendRequest struct {
	ClientMessageID string `json:"client_message_id"`
	Text            string `json:"text"`
}

func (h *Handler) send(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFrom(r.Context())
	if !ok {
		httpapi.WriteError(w, httpapi.Unauthorized("Требуется вход"))
		return
	}
	conversationID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || conversationID <= 0 {
		httpapi.WriteError(w, httpapi.BadRequest("invalid_id", "Некорректный ID"))
		return
	}
	var body sendRequest
	if err := httpapi.DecodeJSON(w, r, &body, 16<<10); err != nil {
		httpapi.WriteError(w, err)
		return
	}

	message, err := h.service.Send(r.Context(), user.ID, conversationID, body.ClientMessageID, body.Text)
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}

	// The signal is a hint to refetch; the message itself is already stored.
	h.events.Publish(conversationID, "chat.changed")
	httpapi.WriteJSON(w, http.StatusCreated, messageJSON(message))
}

type readRequest struct {
	Seq json.Number `json:"seq"`
}

func (h *Handler) markRead(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFrom(r.Context())
	if !ok {
		httpapi.WriteError(w, httpapi.Unauthorized("Требуется вход"))
		return
	}
	conversationID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || conversationID <= 0 {
		httpapi.WriteError(w, httpapi.BadRequest("invalid_id", "Некорректный ID"))
		return
	}
	var body readRequest
	if err := httpapi.DecodeJSON(w, r, &body, 4<<10); err != nil {
		httpapi.WriteError(w, err)
		return
	}
	seq, err := body.Seq.Int64()
	if err != nil {
		httpapi.WriteError(w, httpapi.BadRequest("invalid_seq", "Некорректный номер сообщения"))
		return
	}
	if err := h.service.MarkRead(r.Context(), user.ID, conversationID, seq); err != nil {
		httpapi.WriteError(w, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func parseHistoryQuery(values url.Values) (HistoryQuery, error) {
	query := HistoryQuery{Limit: 50}
	for _, field := range []string{"after", "before"} {
		if !values.Has(field) {
			continue
		}
		seq, err := strconv.ParseInt(values.Get(field), 10, 64)
		if err != nil || seq < 0 || (field == "before" && seq == 0) {
			return query, httpapi.BadRequest("invalid_cursor", "Некорректный курсор истории")
		}
		if field == "after" {
			query.After = &seq
		} else {
			query.Before = &seq
		}
	}
	if query.After != nil && query.Before != nil {
		return query, httpapi.BadRequest("invalid_cursor", "Укажите только один курсор истории")
	}
	if values.Has("limit") {
		limit, err := strconv.Atoi(values.Get("limit"))
		if err != nil || limit < 1 || limit > 100 {
			return query, httpapi.BadRequest("invalid_limit", "Размер страницы должен быть от 1 до 100")
		}
		query.Limit = limit
	}
	return query, nil
}

func conversationJSON(conversation Conversation) map[string]any {
	var partnerID any
	if conversation.PartnerID > 0 {
		partnerID = strconv.FormatInt(conversation.PartnerID, 10)
	}
	var preview any
	if message := conversation.LastMessage; message != nil {
		preview = map[string]any{
			"text":        message.Text,
			"attachment":  message.Attachment,
			"sender_id":   strconv.FormatInt(message.SenderID, 10),
			"sender_name": message.SenderName,
			"created_at":  message.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
		}
	}
	return map[string]any{
		"id": strconv.FormatInt(conversation.ID, 10), "kind": conversation.Kind,
		"title": conversation.Title, "partner_id": partnerID, "last_message": preview,
		"last_seq":  strconv.FormatInt(conversation.LastSeq, 10),
		"last_read": strconv.FormatInt(conversation.LastRead, 10), "unread": conversation.Unread,
		"updated_at": conversation.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
	}
}

func messageJSON(message Message) map[string]any {
	return map[string]any{
		"id":              strconv.FormatInt(message.ID, 10),
		"conversation_id": strconv.FormatInt(message.ConversationID, 10),
		"sender_id":       strconv.FormatInt(message.SenderID, 10),
		"sender_name":     message.SenderName,
		"seq":             strconv.FormatInt(message.Seq, 10),
		"text":            message.Text,
		"attachment":      message.Attachment,
		"created_at":      message.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
	}
}
