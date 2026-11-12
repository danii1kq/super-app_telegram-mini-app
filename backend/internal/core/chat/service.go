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
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

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
	ID          int64
	Kind        string
	Title       string
	LastSeq     int64
	LastRead    int64
	Unread      int
	UpdatedAt   time.Time
	PartnerID   int64
	LastMessage *Message
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
		`SELECT c.id, c.kind, c.title_sealed, c.key_id,
		        c.last_message_seq, m.last_read_seq, COALESCE(latest.created_at, c.created_at),
		        COALESCE(partner.id, 0), COALESCE(partner.public_name, ''),
		        COALESCE(latest.sender_id, 0), COALESCE(latest.public_name, ''),
		        COALESCE(latest.seq, 0), latest.payload_sealed, latest.key_id,
		        (SELECT count(*) FROM chat_messages unread
		          WHERE unread.conversation_id = c.id AND unread.seq > m.last_read_seq
		            AND unread.sender_id <> $1)
		   FROM chat_conversations c
		   JOIN chat_members m ON m.conversation_id = c.id
		   LEFT JOIN LATERAL (
		       SELECT u.id, u.public_name FROM chat_members peer
		       JOIN users u ON u.id = peer.user_id
		       WHERE c.kind = 'direct' AND peer.conversation_id = c.id AND peer.user_id <> $1
		       ORDER BY peer.user_id LIMIT 1
		   ) partner ON true
		   LEFT JOIN LATERAL (
		       SELECT msg.sender_id, u.public_name, msg.seq, msg.payload_sealed, msg.key_id, msg.created_at
		       FROM chat_messages msg JOIN users u ON u.id = msg.sender_id
		       WHERE msg.conversation_id = c.id ORDER BY msg.seq DESC LIMIT 1
		   ) latest ON true
		  WHERE m.user_id = $1 AND m.request_state = 'accepted'
		  ORDER BY COALESCE(latest.created_at, c.created_at) DESC, c.id DESC`, viewerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]Conversation, 0, 8)
	for rows.Next() {
		var conversation Conversation
		var sealed []byte
		var keyID *string
		var partnerName string
		var latest Message
		var latestSealed []byte
		var latestKeyID *string
		if err := rows.Scan(&conversation.ID, &conversation.Kind, &sealed,
			&keyID, &conversation.LastSeq, &conversation.LastRead, &conversation.UpdatedAt,
			&conversation.PartnerID, &partnerName, &latest.SenderID, &latest.SenderName,
			&latest.Seq, &latestSealed, &latestKeyID, &conversation.Unread,
		); err != nil {
			return nil, err
		}
		if len(sealed) > 0 && keyID != nil {
			title, err := s.content.OpenString(secure.Sealed{KeyID: *keyID, Ciphertext: sealed}, titleAAD(conversation.ID))
			if err != nil {
				return nil, err
			}
			conversation.Title = title
		}
		if conversation.Kind == "direct" {
			conversation.Title = partnerName
		}
		if latestKeyID != nil {
			text, err := s.openPayload(latestSealed, *latestKeyID, conversation.ID, latest.Seq)
			if err != nil {
				return nil, err
			}
			latest.ConversationID = conversation.ID
			latest.CreatedAt = conversation.UpdatedAt
			latest.Text = text
			conversation.LastMessage = &latest
		}
		out = append(out, conversation)
	}
	return out, rows.Err()
}

type HistoryQuery struct {
	After  *int64
	Before *int64
	Limit  int
}

// History defaults to the latest page, with ascending display order in every mode.
func (s *Service) History(ctx context.Context, viewerID, conversationID int64, query HistoryQuery) ([]Message, bool, error) {
	if query.Limit < 1 || query.Limit > 100 {
		query.Limit = 50
	}
	if err := s.requireMember(ctx, s.pool, viewerID, conversationID); err != nil {
		return nil, false, err
	}
	order := "DESC"
	if query.After != nil {
		order = "ASC"
	}

	rows, err := s.pool.Query(ctx,
		`SELECT m.id, m.conversation_id, m.sender_id, u.public_name, m.seq,
		        m.payload_sealed, m.key_id, m.created_at
		   FROM chat_messages m
		   JOIN users u ON u.id = m.sender_id
		   JOIN chat_members member ON member.conversation_id = m.conversation_id
		    AND member.user_id = $2 AND member.request_state = 'accepted'
		  WHERE m.conversation_id = $1
		    AND ($3::bigint IS NULL OR m.seq > $3)
		    AND ($4::bigint IS NULL OR m.seq < $4)
		  ORDER BY m.seq `+order+`
		  LIMIT $5`, conversationID, viewerID, query.After, query.Before, query.Limit+1)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()

	out := make([]Message, 0, query.Limit+1)
	for rows.Next() {
		var message Message
		var sealed []byte
		var keyID string
		if err := rows.Scan(&message.ID, &message.ConversationID, &message.SenderID,
			&message.SenderName, &message.Seq, &sealed, &keyID, &message.CreatedAt,
		); err != nil {
			return nil, false, err
		}
		text, err := s.openPayload(sealed, keyID, message.ConversationID, message.Seq)
		if err != nil {
			return nil, false, err
		}
		message.Text = text
		out = append(out, message)
	}
	if err := rows.Err(); err != nil {
		return nil, false, err
	}
	hasMore := len(out) > query.Limit
	if hasMore {
		out = out[:query.Limit]
	}
	if query.After == nil {
		slices.Reverse(out)
	}
	return out, hasMore, nil
}

// Send stores a message idempotently: the same client id returns the stored
// message, a different body with the same id is a conflict.
func (s *Service) Send(ctx context.Context, senderID, conversationID int64, clientID, text string) (Message, error) {
	text = strings.TrimSpace(text)
	if clientID == "" {
		return Message{}, httpapi.BadRequest("missing_client_id", "Не передан client_message_id")
	}
	if len(clientID) > 128 {
		return Message{}, httpapi.BadRequest("invalid_client_id", "Слишком длинный client_message_id")
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
		var blocked bool
		if err := tx.QueryRow(ctx,
			`SELECT EXISTS (
			   SELECT 1 FROM chat_conversations c
			   JOIN chat_members peer ON peer.conversation_id = c.id AND peer.user_id <> $2
			   JOIN social_blocks b ON (b.blocker_id = $2 AND b.blocked_id = peer.user_id)
			                         OR (b.blocker_id = peer.user_id AND b.blocked_id = $2)
			   WHERE c.id = $1 AND c.kind = 'direct')`, conversationID, senderID,
		).Scan(&blocked); err != nil {
			return err
		}
		if blocked {
			return httpapi.NotFound("Диалог не найден")
		}

		// Idempotency: the same sender + client id returns the stored message.
		var existingID, existingSeq, existingConversationID int64
		var existingSealed []byte
		var existingKeyID string
		var existingCreatedAt time.Time
		var senderName string
		err := tx.QueryRow(ctx,
			`SELECT m.id, m.seq, m.conversation_id, m.payload_sealed, m.key_id, m.created_at, u.public_name
			   FROM chat_messages m JOIN users u ON u.id = m.sender_id
			  WHERE m.sender_id = $1 AND m.client_message_id = $2`,
			senderID, clientID,
		).Scan(&existingID, &existingSeq, &existingConversationID, &existingSealed, &existingKeyID, &existingCreatedAt, &senderName)
		if err == nil {
			if existingConversationID != conversationID {
				return httpapi.Conflict("idempotency_conflict", "Этот ключ уже использован в другом диалоге")
			}
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
				CreatedAt: existingCreatedAt, SenderName: senderName,
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
			var conflict *pgconn.PgError
			if errors.As(err, &conflict) && conflict.Code == "23505" {
				return httpapi.Conflict("idempotency_conflict", "Этот ключ уже использован в другом диалоге")
			}
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
		if err := tx.QueryRow(ctx, `SELECT public_name FROM users WHERE id = $1`, senderID).Scan(&message.SenderName); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return Message{}, err
	}
	return message, nil
}

// MarkRead advances last_read_seq, never backwards.
func (s *Service) MarkRead(ctx context.Context, viewerID, conversationID, seq int64) error {
	if seq < 0 {
		return httpapi.BadRequest("invalid_seq", "Некорректный номер сообщения")
	}
	result, err := s.pool.Exec(ctx,
		`UPDATE chat_members m
		    SET last_read_seq = GREATEST(m.last_read_seq, LEAST($3, c.last_message_seq))
		   FROM chat_conversations c
		  WHERE m.conversation_id = c.id AND m.conversation_id = $1
		    AND m.user_id = $2 AND m.request_state = 'accepted'`, conversationID, viewerID, seq)
	if err == nil && result.RowsAffected() == 0 {
		return httpapi.NotFound("Диалог не найден")
	}
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

func titleAAD(conversationID int64) string {
	return "chat_title:" + itoa(conversationID)
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
