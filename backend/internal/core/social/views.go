package social

import (
	"context"
	"errors"

	"github.com/danii1kq/srunal/internal/platform/httpapi"
	"github.com/jackc/pgx/v5"
)

// RecordView counts accounts, not requests. An author's own views never count.
func (s *Service) RecordView(ctx context.Context, viewerID, postID int64) (int, error) {
	var count int
	err := s.pool.InTx(ctx, func(tx pgx.Tx) error {
		var authorID int64
		err := tx.QueryRow(ctx, `SELECT p.author_id FROM social_posts p WHERE `+visiblePostSQL+` AND p.id = $2 FOR SHARE OF p`, viewerID, postID).Scan(&authorID)
		if errors.Is(err, pgx.ErrNoRows) {
			return httpapi.NotFound("Публикация не найдена")
		}
		if err != nil {
			return err
		}
		if viewerID != authorID {
			if _, err := tx.Exec(ctx, `INSERT INTO social_post_views (post_id,user_id)
    SELECT p.id,$1 FROM social_posts p WHERE `+visiblePostSQL+` AND p.id=$2
    ON CONFLICT DO NOTHING`, viewerID, postID); err != nil {
				return err
			}
		}
		return tx.QueryRow(ctx, `SELECT count(*) FROM social_post_views WHERE post_id=$1`, postID).Scan(&count)
	})
	return count, err
}

func (s *Service) AuthorPosts(ctx context.Context, viewerID, authorID, beforeID int64, limit int) ([]Post, error) {
	if _, err := s.Person(ctx, viewerID, authorID); err != nil {
		return nil, err
	}
	return s.posts(ctx, viewerID, authorID, beforeID, limit)
}
