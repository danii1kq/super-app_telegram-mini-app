package pulse

import (
	"context"
	"errors"
	"time"

	"github.com/danii1kq/srunal/internal/platform/httpapi"
	"github.com/jackc/pgx/v5"
)

// Every reader and responder applies the same audience and block rules.
const visibleStatusSQL = `NOT EXISTS (SELECT 1 FROM social_blocks b
 WHERE (b.blocker_id = p.author_id AND b.blocked_id = $1)
    OR (b.blocker_id = $1 AND b.blocked_id = p.author_id))
 AND EXISTS (SELECT 1 FROM users author WHERE author.id = p.author_id AND author.status = 'active')
 AND (p.author_id = $1
  OR EXISTS (SELECT 1 FROM pulse_audience a WHERE a.status_id = p.id AND a.user_id = $1)
  OR EXISTS (SELECT 1 FROM pulse_audience a
    JOIN social_members m ON m.community_id = a.community_id
    JOIN social_communities c ON c.id = a.community_id
    WHERE a.status_id = p.id AND m.user_id = $1 AND m.state = 'active' AND c.state = 'active'))`

func validateAudience(ctx context.Context, tx pgx.Tx, authorID int64, audience Audience) error {
	if len(audience.Users) > 0 {
		var count int
		err := tx.QueryRow(ctx, `SELECT count(*) FROM users u
		 WHERE u.id = ANY($2::bigint[]) AND u.status = 'active'
		 AND NOT EXISTS (SELECT 1 FROM social_blocks b
		   WHERE (b.blocker_id = $1 AND b.blocked_id = u.id)
		      OR (b.blocker_id = u.id AND b.blocked_id = $1))`, authorID, audience.Users).Scan(&count)
		if err != nil {
			return err
		}
		if count != len(audience.Users) {
			return httpapi.Validation(map[string]string{"audience": "Получатель недоступен. Обновите список"}, "Проверьте получателей")
		}
	}
	if audience.Community != nil {
		var member bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS (
		 SELECT 1 FROM social_members m JOIN social_communities c ON c.id = m.community_id
		 WHERE m.community_id = $1 AND m.user_id = $2 AND m.state = 'active' AND c.state = 'active')`,
			*audience.Community, authorID).Scan(&member); err != nil {
			return err
		}
		if !member {
			return httpapi.Forbidden("Нет доступа к сообществу")
		}
	}
	return nil
}

func lockVisibleStatus(ctx context.Context, tx pgx.Tx, viewerID, statusID int64) (int64, time.Time, *time.Time, error) {
	var authorID int64
	var expiresAt time.Time
	var closedAt *time.Time
	err := tx.QueryRow(ctx, `SELECT p.author_id, p.expires_at, p.closed_at FROM pulse_statuses p
	 WHERE p.id = $2 AND `+visibleStatusSQL+` FOR UPDATE OF p`, viewerID, statusID).Scan(&authorID, &expiresAt, &closedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		err = httpapi.NotFound("Статус не найден")
	}
	return authorID, expiresAt, closedAt, err
}
