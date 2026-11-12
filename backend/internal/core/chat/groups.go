package chat

import (
	"context"
	"errors"
	"regexp"
	"strings"

	"github.com/danii1kq/srunal/internal/platform/httpapi"
	"github.com/danii1kq/srunal/internal/platform/secure"
	"github.com/jackc/pgx/v5"
)

const maxGroupMembers = 50

var groupClientID = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

type GroupMember struct {
	UserID     int64
	PublicName string
	Role       string
}

func groupMemberIDs(ownerID int64, selected []int64) ([]int64, error) {
	ids := []int64{ownerID}
	seen := map[int64]bool{ownerID: true}
	for _, id := range selected {
		if id <= 0 {
			return nil, httpapi.BadRequest("invalid_user", "Некорректный пользователь")
		}
		if !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	if len(ids) < 2 || len(ids) > maxGroupMembers {
		return nil, httpapi.Validation(map[string]string{"user_ids": "Выберите от 1 до 49 участников"}, "Проверьте участников")
	}
	return ids, nil
}

func (s *Service) CreateGroup(ctx context.Context, ownerID int64, clientID, title string, selected []int64) (Conversation, error) {
	if !groupClientID.MatchString(clientID) {
		return Conversation{}, httpapi.BadRequest("invalid_client_id", "Не передан корректный client_group_id")
	}
	title = strings.TrimSpace(title)
	if title == "" || len([]rune(title)) > 100 {
		return Conversation{}, httpapi.Validation(map[string]string{"title": "Название должно содержать от 1 до 100 символов"}, "Проверьте название")
	}
	ids, err := groupMemberIDs(ownerID, selected)
	if err != nil {
		return Conversation{}, err
	}
	conversation := Conversation{Kind: "group", Title: title}
	err = s.pool.InTx(ctx, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, `SELECT nextval(pg_get_serial_sequence('chat_conversations', 'id'))`).Scan(&conversation.ID); err != nil {
			return err
		}
		sealed, err := s.content.SealString(title, titleAAD(conversation.ID))
		if err != nil {
			return err
		}
		pairKey := "group:" + itoa(ownerID) + ":" + clientID
		err = tx.QueryRow(ctx, `INSERT INTO chat_conversations (id, kind, title_sealed, key_id, pair_key)
		 OVERRIDING SYSTEM VALUE VALUES ($1, 'group', $2, $3, $4)
		 ON CONFLICT (pair_key) DO NOTHING RETURNING created_at`,
			conversation.ID, sealed.Ciphertext, sealed.KeyID, pairKey).Scan(&conversation.UpdatedAt)
		if errors.Is(err, pgx.ErrNoRows) {
			var stored []byte
			var keyID, role string
			if err := tx.QueryRow(ctx, `SELECT id, title_sealed, key_id, last_message_seq, created_at
			 FROM chat_conversations WHERE pair_key = $1 FOR UPDATE`, pairKey).Scan(&conversation.ID, &stored, &keyID, &conversation.LastSeq, &conversation.UpdatedAt); err != nil {
				return err
			}
			if err := tx.QueryRow(ctx, `SELECT role FROM chat_members
			 WHERE conversation_id = $1 AND user_id = $2 AND request_state = 'accepted'`, conversation.ID, ownerID).Scan(&role); err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return httpapi.NotFound("Группа не найдена")
				}
				return err
			}
			if role != "owner" {
				return httpapi.NotFound("Группа не найдена")
			}
			storedTitle, err := s.content.OpenString(secure.Sealed{KeyID: keyID, Ciphertext: stored}, titleAAD(conversation.ID))
			if err != nil {
				return err
			}
			// shortcut: retries compare current members; persist the initial request if retries must survive membership edits.
			var count, matching int
			if err := tx.QueryRow(ctx, `SELECT count(*), count(*) FILTER (WHERE user_id = ANY($2::bigint[]))
			 FROM chat_members WHERE conversation_id = $1 AND request_state = 'accepted'`, conversation.ID, ids).Scan(&count, &matching); err != nil {
				return err
			}
			if storedTitle != title || count != len(ids) || matching != len(ids) {
				return httpapi.Conflict("idempotency_conflict", "Эта группа уже создана с другим названием или составом")
			}
			return nil
		}
		if err != nil {
			return err
		}
		var active int
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM users WHERE id = ANY($1::bigint[]) AND status = 'active'`, ids).Scan(&active); err != nil {
			return err
		}
		var blocked bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM social_blocks
		 WHERE blocker_id = ANY($1::bigint[]) AND blocked_id = ANY($1::bigint[]))`, ids).Scan(&blocked); err != nil {
			return err
		}
		if active != len(ids) || blocked {
			return httpapi.NotFound("Участник недоступен")
		}
		for _, id := range ids {
			role := "member"
			if id == ownerID {
				role = "owner"
			}
			if _, err := tx.Exec(ctx, `INSERT INTO chat_members (conversation_id, user_id, role) VALUES ($1, $2, $3)`, conversation.ID, id, role); err != nil {
				return err
			}
		}
		return nil
	})
	return conversation, err
}

func (s *Service) GroupMembers(ctx context.Context, viewerID, conversationID int64) ([]GroupMember, error) {
	if err := s.requireMember(ctx, s.pool, viewerID, conversationID); err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx, `SELECT m.user_id, u.public_name, m.role
	 FROM chat_members m JOIN users u ON u.id = m.user_id
	 JOIN chat_conversations c ON c.id = m.conversation_id
	 WHERE c.id = $1 AND c.kind = 'group' AND m.request_state = 'accepted'
	 AND EXISTS (SELECT 1 FROM chat_members viewer WHERE viewer.conversation_id = c.id
	   AND viewer.user_id = $2 AND viewer.request_state = 'accepted')
	 ORDER BY (m.role = 'owner') DESC, u.public_name, m.user_id`, conversationID, viewerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var members []GroupMember
	for rows.Next() {
		var member GroupMember
		if err := rows.Scan(&member.UserID, &member.PublicName, &member.Role); err != nil {
			return nil, err
		}
		members = append(members, member)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(members) == 0 {
		return nil, httpapi.NotFound("Группа не найдена")
	}
	return members, nil
}

// Membership mutations share Send's conversation lock so removal revokes new sends.
func (s *Service) lockGroup(ctx context.Context, tx pgx.Tx, viewerID, conversationID int64) (string, int64, error) {
	var kind, role string
	var seq int64
	err := tx.QueryRow(ctx, `SELECT kind, last_message_seq FROM chat_conversations
	 WHERE id = $1 FOR UPDATE`, conversationID).Scan(&kind, &seq)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && kind != "group") {
		return "", 0, httpapi.NotFound("Группа не найдена")
	}
	if err != nil {
		return "", 0, err
	}
	err = tx.QueryRow(ctx, `SELECT role FROM chat_members
	 WHERE conversation_id = $1 AND user_id = $2 AND request_state = 'accepted'`, conversationID, viewerID).Scan(&role)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", 0, httpapi.NotFound("Группа не найдена")
	}
	return role, seq, err
}

func (s *Service) AddGroupMember(ctx context.Context, viewerID, conversationID, userID int64) error {
	if userID <= 0 {
		return httpapi.BadRequest("invalid_user", "Некорректный пользователь")
	}
	return s.pool.InTx(ctx, func(tx pgx.Tx) error {
		role, seq, err := s.lockGroup(ctx, tx, viewerID, conversationID)
		if err != nil {
			return err
		}
		if role != "owner" {
			return httpapi.Forbidden("Только владелец может добавлять участников")
		}
		var exists bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM chat_members
		 WHERE conversation_id = $1 AND user_id = $2 AND request_state = 'accepted')`, conversationID, userID).Scan(&exists); err != nil {
			return err
		}
		if exists {
			return nil
		}
		var count int
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM chat_members WHERE conversation_id = $1 AND request_state = 'accepted'`, conversationID).Scan(&count); err != nil {
			return err
		}
		if count >= maxGroupMembers {
			return httpapi.Conflict("group_full", "В группе уже 50 участников")
		}
		var active, blocked bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM users WHERE id = $1 AND status = 'active'),
		 EXISTS (SELECT 1 FROM social_blocks b JOIN chat_members m
		   ON (b.blocker_id = $1 AND b.blocked_id = m.user_id)
		   OR (b.blocked_id = $1 AND b.blocker_id = m.user_id)
		   WHERE m.conversation_id = $2 AND m.request_state = 'accepted')`, userID, conversationID).Scan(&active, &blocked); err != nil {
			return err
		}
		if !active || blocked {
			return httpapi.NotFound("Участник недоступен")
		}
		_, err = tx.Exec(ctx, `INSERT INTO chat_members (conversation_id, user_id, last_read_seq) VALUES ($1, $2, $3)
		 ON CONFLICT (conversation_id, user_id) DO UPDATE SET role = 'member', request_state = 'accepted',
		 last_read_seq = excluded.last_read_seq, joined_at = now()`, conversationID, userID, seq)
		return err
	})
}

func (s *Service) RemoveGroupMember(ctx context.Context, viewerID, conversationID, userID int64) error {
	if userID <= 0 {
		return httpapi.BadRequest("invalid_user", "Некорректный пользователь")
	}
	return s.pool.InTx(ctx, func(tx pgx.Tx) error {
		role, _, err := s.lockGroup(ctx, tx, viewerID, conversationID)
		if err != nil {
			return err
		}
		if userID != viewerID && role != "owner" {
			return httpapi.Forbidden("Только владелец может удалять участников")
		}
		if userID == viewerID && role == "owner" {
			var others bool
			if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM chat_members
			 WHERE conversation_id = $1 AND user_id <> $2 AND request_state = 'accepted')`, conversationID, viewerID).Scan(&others); err != nil {
				return err
			}
			if others {
				return httpapi.Conflict("owner_must_remove_members", "Сначала удалите остальных участников группы")
			}
		}
		_, err = tx.Exec(ctx, `DELETE FROM chat_members WHERE conversation_id = $1 AND user_id = $2`, conversationID, userID)
		return err
	})
}
