package pulse

import (
	"context"

	"github.com/danii1kq/srunal/internal/platform/httpapi"
	"github.com/jackc/pgx/v5"
)

type Responder struct {
	ID   int64
	Name string
}

// The author may see who responded; recipients only receive the total count.
func (s *Service) Responders(ctx context.Context, authorID, statusID int64) ([]Responder, error) {
	var mine bool
	if err := s.pool.QueryRow(ctx, `SELECT EXISTS (
	 SELECT 1 FROM pulse_statuses WHERE id = $2 AND author_id = $1)`, authorID, statusID).Scan(&mine); err != nil {
		return nil, err
	}
	if !mine {
		return nil, httpapi.NotFound("Статус не найден")
	}
	// shortcut: show the first 100 responders; add pagination for larger communities.
	rows, err := s.pool.Query(ctx, `SELECT u.id, u.public_name
	 FROM pulse_responses r JOIN users u ON u.id = r.user_id
	 WHERE r.status_id = $2 AND u.status = 'active'
	 AND NOT EXISTS (SELECT 1 FROM social_blocks b
	  WHERE (b.blocker_id = $1 AND b.blocked_id = u.id)
	     OR (b.blocker_id = u.id AND b.blocked_id = $1))
	 ORDER BY r.created_at, r.user_id LIMIT 100`, authorID, statusID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Responder, 0)
	for rows.Next() {
		var responder Responder
		if err := rows.Scan(&responder.ID, &responder.Name); err != nil {
			return nil, err
		}
		out = append(out, responder)
	}
	return out, rows.Err()
}

func (s *Service) CancelResponse(ctx context.Context, viewerID, statusID int64) error {
	return s.pool.InTx(ctx, func(tx pgx.Tx) error {
		if _, _, _, err := lockVisibleStatus(ctx, tx, viewerID, statusID); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `DELETE FROM pulse_responses WHERE status_id = $1 AND user_id = $2`, statusID, viewerID)
		return err
	})
}
