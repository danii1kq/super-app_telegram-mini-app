package chat

import (
	"net/http"
	"strconv"

	"github.com/danii1kq/srunal/internal/core/auth"
	"github.com/danii1kq/srunal/internal/platform/httpapi"
)

func (h *Handler) registerGroups(mux *http.ServeMux) {
	mux.Handle("POST /api/v1/conversations/groups", h.auth.Middleware(http.HandlerFunc(h.createGroup)))
	mux.Handle("GET /api/v1/conversations/{id}/members", h.auth.Middleware(http.HandlerFunc(h.groupMembers)))
	mux.Handle("POST /api/v1/conversations/{id}/members", h.auth.Middleware(http.HandlerFunc(h.addGroupMember)))
	mux.Handle("DELETE /api/v1/conversations/{id}/members/{userId}", h.auth.Middleware(http.HandlerFunc(h.removeGroupMember)))
}

func positiveID(raw string) (int64, error) {
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 || strconv.FormatInt(id, 10) != raw {
		return 0, httpapi.BadRequest("invalid_id", "Некорректный ID")
	}
	return id, nil
}

func (h *Handler) createGroup(w http.ResponseWriter, r *http.Request) {
	viewer, ok := auth.UserFrom(r.Context())
	if !ok {
		httpapi.WriteError(w, httpapi.Unauthorized("Требуется вход"))
		return
	}
	var request struct {
		ClientGroupID string   `json:"client_group_id"`
		Title         string   `json:"title"`
		UserIDs       []string `json:"user_ids"`
	}
	if err := httpapi.DecodeJSON(w, r, &request, 16<<10); err != nil {
		httpapi.WriteError(w, err)
		return
	}
	ids := make([]int64, 0, len(request.UserIDs))
	for _, raw := range request.UserIDs {
		id, err := positiveID(raw)
		if err != nil {
			httpapi.WriteError(w, err)
			return
		}
		ids = append(ids, id)
	}
	conversation, err := h.service.CreateGroup(r.Context(), viewer.ID, request.ClientGroupID, request.Title, ids)
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusCreated, conversationJSON(conversation))
}

func (h *Handler) groupMembers(w http.ResponseWriter, r *http.Request) {
	viewer, ok := auth.UserFrom(r.Context())
	if !ok {
		httpapi.WriteError(w, httpapi.Unauthorized("Требуется вход"))
		return
	}
	id, err := positiveID(r.PathValue("id"))
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}
	members, err := h.service.GroupMembers(r.Context(), viewer.ID, id)
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}
	out := make([]map[string]any, 0, len(members))
	for _, member := range members {
		out = append(out, map[string]any{"user_id": strconv.FormatInt(member.UserID, 10), "public_name": member.PublicName, "role": member.Role})
	}
	httpapi.WriteJSON(w, http.StatusOK, map[string]any{"members": out})
}

func (h *Handler) addGroupMember(w http.ResponseWriter, r *http.Request) {
	viewer, ok := auth.UserFrom(r.Context())
	if !ok {
		httpapi.WriteError(w, httpapi.Unauthorized("Требуется вход"))
		return
	}
	id, err := positiveID(r.PathValue("id"))
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}
	var request struct {
		UserID string `json:"user_id"`
	}
	if err := httpapi.DecodeJSON(w, r, &request, 1<<10); err != nil {
		httpapi.WriteError(w, err)
		return
	}
	userID, err := positiveID(request.UserID)
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}
	if err := h.service.AddGroupMember(r.Context(), viewer.ID, id, userID); err != nil {
		httpapi.WriteError(w, err)
		return
	}
	h.events.Publish(id, "chat.changed")
	httpapi.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *Handler) removeGroupMember(w http.ResponseWriter, r *http.Request) {
	viewer, ok := auth.UserFrom(r.Context())
	if !ok {
		httpapi.WriteError(w, httpapi.Unauthorized("Требуется вход"))
		return
	}
	id, err := positiveID(r.PathValue("id"))
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}
	userID, err := positiveID(r.PathValue("userId"))
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}
	if err := h.service.RemoveGroupMember(r.Context(), viewer.ID, id, userID); err != nil {
		httpapi.WriteError(w, err)
		return
	}
	h.events.Publish(id, "chat.changed")
	httpapi.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
