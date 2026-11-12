package chat

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/danii1kq/srunal/internal/platform/httpapi"
)

// OpenDirect opens the unique pair without a select/insert race.
func (s *Service) OpenDirect(ctx context.Context, viewerID, otherID int64) (Conversation, error) {
	if viewerID == otherID || otherID <= 0 {
		return Conversation{}, httpapi.Validation(map[string]string{"user_id": "Выберите другого пользователя"}, "Проверьте поля")
	}
	var blocked bool
	if err := s.pool.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM social_blocks
                         WHERE (blocker_id = $1 AND blocked_id = $2)
                            OR (blocker_id = $2 AND blocked_id = $1))`, viewerID, otherID,
	).Scan(&blocked); err != nil {
		return Conversation{}, err
	}
	if blocked {
		return Conversation{}, httpapi.NotFound("Диалог не найден")
	}
	var otherName string
	if err := s.pool.QueryRow(ctx,
		`SELECT public_name FROM users WHERE id = $1 AND status = 'active'`, otherID,
	).Scan(&otherName); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Conversation{}, httpapi.NotFound("Пользователь не найден")
		}
		return Conversation{}, err
	}
	low, high := viewerID, otherID
	if low > high {
		low, high = high, low
	}
	pairKey := "direct:" + itoa(low) + ":" + itoa(high)
	conversation := Conversation{Kind: "direct", Title: otherName, PartnerID: otherID}
	err := s.pool.InTx(ctx, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx,
			`INSERT INTO chat_conversations (kind, pair_key) VALUES ('direct', $1)
             ON CONFLICT (pair_key) DO UPDATE SET pair_key = excluded.pair_key
             RETURNING id, last_message_seq, created_at`, pairKey,
		).Scan(&conversation.ID, &conversation.LastSeq, &conversation.UpdatedAt); err != nil {
			return err
		}
		for _, member := range []int64{viewerID, otherID} {
			if _, err := tx.Exec(ctx,
				`INSERT INTO chat_members (conversation_id, user_id) VALUES ($1, $2)
                 ON CONFLICT DO NOTHING`, conversation.ID, member,
			); err != nil {
				return err
			}
		}
		return tx.QueryRow(ctx,
			`SELECT last_read_seq FROM chat_members WHERE conversation_id = $1 AND user_id = $2`,
			conversation.ID, viewerID,
		).Scan(&conversation.LastRead)
	})
	if err != nil {
		return Conversation{}, err
	}
	return conversation, nil
}
