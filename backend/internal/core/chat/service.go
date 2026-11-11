// Package chat owns conversations, membership and messages.
//
// Message order is a per-conversation seq: the send transaction locks the
// conversation row, re-checks membership, increments last_message_seq and
// stores the message, so a cursor can never skip a message that committed late.
package chat

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/danii1kq/srunal/internal/platform/httpapi"
	"github.com/danii1kq/srunal/internal/platform/postgres"
	"github.com/danii1kq/srunal/internal/platform/secure"
)

type Service struct {
	pool    *postgres.Pool
	content *secure.Cipher
}

func NewService(pool *postgres.Pool, content *secure.Cipher) *Service {
	return &Service{pool: pool, content: content}
}

type Conversation struct {
	ID        int64
	Kind      string
	Title     string
	LastSeq   int64
	LastRead  int64
	Unread    int
	UpdatedAt time.Time
}

type Message struct {
	ID             int64
	ConversationID int64
	SenderID       int64
	SenderName     string
	Seq            int64
	Text           string
	CreatedAt      time.Time
}

// Payload is the sealed message body. Text is the only kind in the first
// version; the struct leaves room for a card without changing the table.
type Payload struct {
	Kind string `json:"kind"`
	Text string `json:"text"`
}

// List returns the viewer's conversations with unread counts.
func (s *Service) List(ctx context.Context, viewerID int64) ([]Conversation, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT c.id, c.kind, coalesce(c.title_sealed, ''), c.key_id,
		        c.last_message_seq, m.last_read_seq, c.created_at
		   FROM chat_conversations c
		   JOIN chat_members m ON m.conversation_id = c.id
		  WHERE m.user_id = $1 AND m.request_state = 'accepted'
		  ORDER BY c.last_message_seq DESC, c.id DESC`, viewerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]Conversation, 0, 8)
	for rows.Next() {
		var conversation Conversation
		var sealed []byte
		var keyID *string
		if err := rows.Scan(&conversation.ID, &conversation.Kind, &conversation.Title,
			&keyID, &conversation.LastSeq, &conversation.LastRead, &conversation.UpdatedAt,
		); err != nil {
			return nil, err
		}
		_ = sealed
		if keyID != nil && conversation.Title != "" {
			// Title is stored sealed for groups; the placeholder above keeps the
			// scan simple until the title column is read separately.
			conversation.Title = ""
		}
		if conversation.LastSeq > conversation.LastRead {
			conversation.Unread = int(conversation.LastSeq - conversation.LastRead)
		}
		out = append(out, conversation)
	}
	return out, rows.Err()
}

// History returns messages after the given seq, oldest first.
func (s *Service) History(ctx context.Context, viewerID, conversationID, afterSeq int64, limit int) ([]Message, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if err := s.requireMember(ctx, s.pool, viewerID, conversationID); err != nil {
		return nil, err
	}

	rows, err := s.pool.Query(ctx,
		`SELECT m.id, m.conversation_id, m.sender_id, u.public_name, m.seq,
		        m.payload_sealed, m.key_id, m.created_at
		   FROM chat_messages m
		   JOIN users u ON u.id = m.sender_id
		  WHERE m.conversation_id = $1 AND m.seq > $2
		  ORDER BY m.seq
		  LIMIT $3`, conversationID, afterSeq, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]Message, 0, limit)
	for rows.Next() {
		var message Message
		var sealed []byte
		var keyID string
		if err := rows.Scan(&message.ID, &message.ConversationID, &message.SenderID,
			&message.SenderName, &message.Seq, &sealed, &keyID, &message.CreatedAt,
		); err != nil {
			return nil, err
		}
		text, err := s.openPayload(sealed, keyID, message.ConversationID, message.Seq)
		if err != nil {
			return nil, err
		}
		message.Text = text
		out = append(out, message)
	}
	return out, rows.Err()
}

// Send stores a message idempotently: the same client id returns the stored
// message, a different body with the same id is a conflict.
func (s *Service) Send(ctx context.Context, senderID, conversationID int64, clientID, text string) (Message, error) {
	if clientID == "" {
		return Message{}, httpapi.BadRequest("missing_client_id", "Не передан client_message_id")
	}
	if text == "" {
		return Message{}, httpapi.Validation(map[string]string{"text": "Сообщение не может быть пустым"}, "Проверьте поля")
	}
	if len([]rune(text)) > 4000 {
		return Message{}, httpapi.Validation(map[string]string{"text": "Слишком длинное сообщение"}, "Проверьте поля")
	}

	var message Message
	err := s.pool.InTx(ctx, func(tx pgx.Tx) error {
		// Lock the conversation row: membership and send are serialised here.
		var lastSeq int64
		if err := tx.QueryRow(ctx,
			`SELECT last_message_seq FROM chat_conversations WHERE id = $1 FOR UPDATE`,
			conversationID,
		).Scan(&lastSeq); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return httpapi.NotFound("Диалог не найден")
			}
			return err
		}

		if err := s.requireMember(ctx, tx, senderID, conversationID); err != nil {
			return err
		}

		// Idempotency: the same sender + client id returns the stored message.
		var existingID, existingSeq int64
		var existingSealed []byte
		var existingKeyID string
		err := tx.QueryRow(ctx,
			`SELECT id, seq, payload_sealed, key_id FROM chat_messages
			  WHERE sender_id = $1 AND client_message_id = $2`,
			senderID, clientID,
		).Scan(&existingID, &existingSeq, &existingSealed, &existingKeyID)
		if err == nil {
			existingText, err := s.openPayload(existingSealed, existingKeyID, conversationID, existingSeq)
			if err != nil {
				return err
			}
			if existingText != text {
				return httpapi.Conflict("idempotency_conflict", "Этот ключ уже использован с другим сообщением")
			}
			message = Message{
				ID: existingID, ConversationID: conversationID, SenderID: senderID,
				Seq: existingSeq, Text: existingText,
			}
			return nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}

		nextSeq := lastSeq + 1
		payload, err := json.Marshal(Payload{Kind: "text", Text: text})
		if err != nil {
			return err
		}
		sealed, err := s.content.Seal(payload, messageAAD(conversationID, nextSeq))
		if err != nil {
			return err
		}

		if err := tx.QueryRow(ctx,
			`INSERT INTO chat_messages (conversation_id, sender_id, seq, client_message_id, payload_sealed, key_id)
			 VALUES ($1, $2, $3, $4, $5, $6)
			 RETURNING id, created_at`,
			conversationID, senderID, nextSeq, clientID, sealed.Ciphertext, sealed.KeyID,
		).Scan(&message.ID, &message.CreatedAt); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx,
			`UPDATE chat_conversations SET last_message_seq = $2 WHERE id = $1`,
			conversationID, nextSeq,
		); err != nil {
			return err
		}

		message.ConversationID = conversationID
		message.SenderID = senderID
		message.Seq = nextSeq
		message.Text = text
		return nil
	})
	if err != nil {
		return Message{}, err
	}
	return message, nil
}

// MarkRead advances last_read_seq, never backwards.
func (s *Service) MarkRead(ctx context.Context, viewerID, conversationID, seq int64) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE chat_members SET last_read_seq = GREATEST(last_read_seq, $3)
		  WHERE conversation_id = $1 AND user_id = $2`, conversationID, viewerID, seq)
	return err
}

// querier is the shared subset of pgxpool and pgx.Tx.
type querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func (s *Service) requireMember(ctx context.Context, q querier, userID, conversationID int64) error {
	var accepted bool
	err := q.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM chat_members
		                 WHERE conversation_id = $1 AND user_id = $2 AND request_state = 'accepted')`,
		conversationID, userID,
	).Scan(&accepted)
	if err != nil {
		return err
	}
	if !accepted {
		// Neutral answer: a stranger learns nothing about the conversation.
		return httpapi.NotFound("Диалог не найден")
	}
	return nil
}

func (s *Service) openPayload(sealed []byte, keyID string, conversationID, seq int64) (string, error) {
	raw, err := s.content.Open(secure.Sealed{KeyID: keyID, Ciphertext: sealed}, messageAAD(conversationID, seq))
	if err != nil {
		return "", err
	}
	var payload Payload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return "", err
	}
	return payload.Text, nil
}

func messageAAD(conversationID, seq int64) string {
	return "chat_message:" + itoa(conversationID) + ":" + itoa(seq)
}

func itoa(value int64) string {
	if value == 0 {
		return "0"
	}
	var buffer [20]byte
	position := len(buffer)
	negative := value < 0
	if negative {
		value = -value
	}
	for value > 0 {
		position--
		buffer[position] = byte('0' + value%10)
		value /= 10
	}
	if negative {
		position--
		buffer[position] = '-'
	}
	return string(buffer[position:])
}
