package chat

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"image"
	"path"
	"strings"
	"unicode"

	"github.com/danii1kq/srunal/internal/platform/httpapi"
	"github.com/danii1kq/srunal/internal/platform/media"
	"github.com/danii1kq/srunal/internal/platform/secure"
	"github.com/jackc/pgx/v5"
)

const MaxAttachmentBytes = 20 << 20

type Attachment struct {
	Kind   string `json:"kind"`
	Name   string `json:"name"`
	Mime   string `json:"mime"`
	Size   int    `json:"size"`
	SHA256 string `json:"sha256"`
	Width  int    `json:"width,omitempty"`
	Height int    `json:"height,omitempty"`
}

func prepareAttachment(kind, name string, raw []byte) (*Attachment, []byte, error) {
	if len(raw) == 0 || len(raw) > MaxAttachmentBytes {
		return nil, nil, httpapi.BadRequest("file_size", "Файл должен быть от 1 байта до 20 МБ")
	}
	if kind != "photo" && kind != "file" {
		return nil, nil, httpapi.BadRequest("file_kind", "Выберите фото или файл")
	}
	name = path.Base(strings.ReplaceAll(name, "\\", "/"))
	name = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || unicode.In(r, unicode.Cf) || r == ':' {
			return -1
		}
		return r
	}, name)
	name = string([]rune(name)[:min(len([]rune(name)), 180)])
	if name == "" || name == "." {
		name = "file"
	}
	info := &Attachment{Kind: kind, Name: name, Mime: "application/octet-stream"}
	if kind == "photo" {
		config, _, err := image.DecodeConfig(bytes.NewReader(raw))
		if err != nil || int64(config.Width)*int64(config.Height) > 16_000_000 {
			return nil, nil, httpapi.BadRequest("invalid_photo", "Выберите JPEG, PNG или WebP до 16 мегапикселей")
		}
		result, err := media.Normalize(raw)
		if err != nil {
			return nil, nil, httpapi.BadRequest("invalid_photo", "Не удалось прочитать фотографию")
		}
		raw = result.Data
		info.Name, info.Mime, info.Width, info.Height = "photo.jpg", result.Mime, result.Width, result.Height
	}
	if len(raw) > MaxAttachmentBytes {
		return nil, nil, httpapi.BadRequest("file_size", "Файл превышает 20 МБ")
	}
	sum := sha256.Sum256(raw)
	info.Size, info.SHA256 = len(raw), hex.EncodeToString(sum[:])
	return info, raw, nil
}

func (s *Service) SendAttachment(ctx context.Context, senderID, conversationID int64, clientID, text, kind, name string, raw []byte) (Message, error) {
	if err := s.requireMember(ctx, s.pool, senderID, conversationID); err != nil {
		return Message{}, err
	}
	info, data, err := prepareAttachment(kind, name, raw)
	if err != nil {
		return Message{}, err
	}
	return s.send(ctx, senderID, conversationID, clientID, Payload{Kind: kind, Text: text, Attachment: info}, data)
}

func (s *Service) DownloadAttachment(ctx context.Context, viewerID, conversationID, messageID int64) ([]byte, *Attachment, error) {
	var sealed, payload []byte
	var keyID, payloadKey string
	var seq int64
	err := s.pool.QueryRow(ctx, `SELECT a.body_sealed,a.key_id,m.payload_sealed,m.key_id,m.seq
 FROM chat_attachments a JOIN chat_messages m ON m.id=a.message_id
 JOIN chat_members member ON member.conversation_id=m.conversation_id
 WHERE m.id=$1 AND m.conversation_id=$2 AND member.user_id=$3 AND member.request_state='accepted'`, messageID, conversationID, viewerID).Scan(&sealed, &keyID, &payload, &payloadKey, &seq)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, httpapi.NotFound("Вложение не найдено")
	}
	if err != nil {
		return nil, nil, err
	}
	body, err := s.decodePayload(payload, payloadKey, conversationID, seq)
	if err != nil {
		return nil, nil, err
	}
	if body.Attachment == nil {
		return nil, nil, httpapi.NotFound("Вложение не найдено")
	}
	raw, err := s.content.Open(secure.Sealed{KeyID: keyID, Ciphertext: sealed}, attachmentAAD(conversationID, messageID))
	return raw, body.Attachment, err
}

func attachmentAAD(conversationID, messageID int64) string {
	return "chat_attachment:" + itoa(conversationID) + ":" + itoa(messageID)
}
