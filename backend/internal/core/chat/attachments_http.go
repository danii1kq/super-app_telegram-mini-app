package chat

import (
	"io"
	"mime"
	"net/http"
	"strconv"
	"time"

	"github.com/danii1kq/srunal/internal/core/auth"
	"github.com/danii1kq/srunal/internal/platform/httpapi"
)

func (h *Handler) registerAttachments(mux *http.ServeMux) {
	mux.Handle("POST /api/v1/conversations/{id}/attachments", h.auth.Middleware(http.HandlerFunc(h.uploadAttachment)))
	mux.Handle("GET /api/v1/conversations/{id}/messages/{messageId}/attachment", h.auth.Middleware(http.HandlerFunc(h.downloadAttachment)))
}

func (h *Handler) uploadAttachment(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFrom(r.Context())
	conversationID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || conversationID <= 0 {
		httpapi.WriteError(w, httpapi.BadRequest("invalid_id", "Некорректный диалог"))
		return
	}
	if err := h.service.requireMember(r.Context(), h.service.pool, user.ID, conversationID); err != nil {
		httpapi.WriteError(w, err)
		return
	}
	select {
	case h.service.uploadSlots <- struct{}{}:
		defer func() { <-h.service.uploadSlots }()
	default:
		httpapi.WriteError(w, &httpapi.Error{Status: http.StatusServiceUnavailable, Code: "upload_busy", Message: "Загрузка занята, попробуйте ещё раз"})
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, MaxAttachmentBytes+(64<<10))
	reader, err := r.MultipartReader()
	if err != nil {
		httpapi.WriteError(w, httpapi.BadRequest("invalid_upload", "Ожидается файл"))
		return
	}
	fields := map[string]string{}
	var raw []byte
	var filename string
	for count := 0; ; count++ {
		part, readErr := reader.NextPart()
		if readErr == io.EOF {
			break
		}
		if readErr != nil || count >= 4 {
			httpapi.WriteError(w, httpapi.BadRequest("invalid_upload", "Неверный размер или формат загрузки"))
			return
		}
		name := part.FormName()
		if name == "file" {
			if raw != nil {
				httpapi.WriteError(w, httpapi.BadRequest("invalid_upload", "Отправьте один файл"))
				return
			}
			raw, err = io.ReadAll(io.LimitReader(part, MaxAttachmentBytes+1))
			filename = part.FileName()
		} else {
			if _, exists := fields[name]; exists || (name != "kind" && name != "text" && name != "client_message_id") {
				httpapi.WriteError(w, httpapi.BadRequest("invalid_upload", "Неверное поле загрузки"))
				return
			}
			var data []byte
			data, err = io.ReadAll(io.LimitReader(part, 16001))
			if len(data) > 16000 {
				httpapi.WriteError(w, httpapi.BadRequest("invalid_upload", "Слишком длинное поле"))
				return
			}
			fields[name] = string(data)
		}
		if err != nil {
			httpapi.WriteError(w, httpapi.BadRequest("invalid_upload", "Не удалось загрузить файл"))
			return
		}
	}
	message, err := h.service.SendAttachment(r.Context(), user.ID, conversationID, fields["client_message_id"], fields["text"], fields["kind"], filename, raw)
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}
	h.events.Publish(conversationID, "chat.changed")
	httpapi.WriteJSON(w, http.StatusCreated, messageJSON(message))
}

func (h *Handler) downloadAttachment(w http.ResponseWriter, r *http.Request) {
	select {
	case h.service.downloadSlots <- struct{}{}:
		defer func() { <-h.service.downloadSlots }()
	case <-r.Context().Done():
		return
	}
	_ = http.NewResponseController(w).SetWriteDeadline(time.Now().Add(30 * time.Second))
	user, _ := auth.UserFrom(r.Context())
	conversationID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	messageID, messageErr := strconv.ParseInt(r.PathValue("messageId"), 10, 64)
	if err != nil || messageErr != nil || conversationID <= 0 || messageID <= 0 {
		httpapi.WriteError(w, httpapi.NotFound("Вложение не найдено"))
		return
	}
	raw, info, err := h.service.DownloadAttachment(r.Context(), user.ID, conversationID, messageID)
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}
	disposition := "attachment"
	if info.Kind == "photo" {
		disposition = "inline"
	}
	w.Header().Set("Content-Type", info.Mime)
	w.Header().Set("Content-Disposition", mime.FormatMediaType(disposition, map[string]string{"filename": info.Name}))
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", "sandbox; default-src 'none'")
	w.Header().Set("Content-Length", strconv.Itoa(len(raw)))
	w.Write(raw)
}
