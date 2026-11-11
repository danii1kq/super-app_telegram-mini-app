package chat

import (
	"context"
	"errors"
	"strconv"

	"github.com/jackc/pgx/v5"

	"github.com/danii1kq/srunal/internal/platform/httpapi"
)

// OpenDirect returns the one-to-one conversation with the other user, creating
// it on first use. pair_key makes it idempotent: two people tapping "write" at
// the same moment still end up in one dialog, not two.
func (s *Service) OpenDirect(ctx context.Context, viewerID, otherID int64) (Conversation, error) {
	if viewerID == otherID {
		return Conversation{}, httpapi.Validation(
			map[string]string{"user_id": "Нельзя открыть диалог с собой"},
			"Проверьте поля",
		)
	}

	var blocked bool
	if err := s.pool.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM social_blocks
		                 WHERE (blocker_id = $1 AND blocked_id = $2)
		                    OR (blocker_id = $2 AND blocked_id = $1))`,
		viewerID, otherID,
	).Scan(&blocked); err != nil {
		return Conversation{}, err
	}
	if blocked {
		// Neutral: a block must not reveal itself through the chat.
		return Conversation{}, httpapi.NotFound("Диалог не найден")
	}

	var otherActive bool
	if err := s.pool.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM users WHERE id = $1 AND status = 'active')`, otherID,
	).Scan(&otherActive); err != nil {
		return Conversation{}, err
	}
	if !otherActive {
		return Conversation{}, httpapi.NotFound("Пользователь не найден")
	}

	low, high := viewerID, otherID
	if low > high {
		low, high = high, low
	}
	pairKey := "direct:" + itoa(low) + ":" + itoa(high)

	var conversation Conversation
	err := s.pool.InTx(ctx, func(tx pgx.Tx) error {
		var id int64
		err := tx.QueryRow(ctx,
			`SELECT id FROM chat_conversations WHERE pair_key = $1`, pairKey,
		).Scan(&id)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return err
		}

		if errors.Is(err, pgx.ErrNoRows) {
			if err := tx.QueryRow(ctx,
				`INSERT INTO chat_conversations (kind, pair_key) VALUES ('direct', $1) RETURNING id`,
				pairKey,
			).Scan(&id); err != nil {
				return err
			}
			for _, member := range []int64{viewerID, otherID} {
				if _, err := tx.Exec(ctx,
					`INSERT INTO chat_members (conversation_id, user_id) VALUES ($1, $2)
					 ON CONFLICT DO NOTHING`, id, member,
				); err != nil {
					return err
				}
			}
		} else {
			// Both sides must be members: a dialog created before a block was
			// lifted still has to accept the missing side.
			for _, member := range []int64{viewerID, otherID} {
				if _, err := tx.Exec(ctx,
					`INSERT INTO chat_members (conversation_id, user_id) VALUES ($1, $2)
					 ON CONFLICT DO NOTHING`, id, member,
				); err != nil {
					return err
				}
			}
		}

		conversation = Conversation{ID: id, Kind: "direct"}
		return tx.QueryRow(ctx,
			`SELECT last_message_seq, created_at FROM chat_conversations WHERE id = $1`, id,
		).Scan(&conversation.LastSeq, &conversation.UpdatedAt)
	})
	if err != nil {
		return Conversation{}, err
	}
	return conversation, nil
}

func formatID(value int64) string {
	return strconv.FormatInt(value, 10)
}
