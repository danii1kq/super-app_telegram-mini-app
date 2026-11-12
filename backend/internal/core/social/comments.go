package social

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/danii1kq/srunal/internal/platform/httpapi"
	"github.com/danii1kq/srunal/internal/platform/secure"
	"github.com/jackc/pgx/v5"
)

type Comment struct {
	ID         int64
	PostID     int64
	AuthorID   int64
	AuthorName string
	Body       string
	CreatedAt  time.Time
}

// Comments returns the next page, newest first after checking post access.
func (s *Service) Comments(ctx context.Context, viewerID, postID, beforeID int64) ([]Comment, error) {
	if _, err := s.byID(ctx, postID, viewerID); err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx, `SELECT c.id, c.author_id, u.public_name,
	 coalesce(c.body_public, ''), c.body_sealed, c.key_id, c.created_at
	 FROM social_comments c JOIN social_posts p ON p.id = c.post_id
	 JOIN users u ON u.id = c.author_id
	 WHERE `+visiblePostSQL+` AND c.post_id = $2 AND ($3 = 0 OR c.id < $3) AND c.deleted_at IS NULL
	 AND NOT EXISTS (SELECT 1 FROM social_blocks b
	   WHERE (b.blocker_id = c.author_id AND b.blocked_id = $1)
	      OR (b.blocker_id = $1 AND b.blocked_id = c.author_id))
	 ORDER BY c.id DESC LIMIT 21`, viewerID, postID, beforeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Comment, 0, 21)
	for rows.Next() {
		comment := Comment{PostID: postID}
		var sealed []byte
		var keyID *string
		if err := rows.Scan(&comment.ID, &comment.AuthorID, &comment.AuthorName, &comment.Body, &sealed, &keyID, &comment.CreatedAt); err != nil {
			return nil, err
		}
		if len(sealed) > 0 && keyID != nil {
			comment.Body, err = s.content.OpenString(secure.Sealed{KeyID: *keyID, Ciphertext: sealed}, commentAAD(comment.ID, comment.AuthorID))
			if err != nil {
				return nil, err
			}
		}
		out = append(out, comment)
	}
	return out, rows.Err()
}

func (s *Service) CreateComment(ctx context.Context, authorID, postID int64, body string) (Comment, error) {
	body = strings.TrimSpace(body)
	if body == "" || len([]rune(body)) > 2000 {
		return Comment{}, httpapi.Validation(map[string]string{"body": "Введите комментарий до 2000 символов"}, "Проверьте текст")
	}
	comment := Comment{PostID: postID, AuthorID: authorID, Body: body}
	err := s.pool.InTx(ctx, func(tx pgx.Tx) error {
		var audience string
		if err := tx.QueryRow(ctx, `SELECT p.audience FROM social_posts p
		 WHERE `+visiblePostSQL+` AND p.id = $2 FOR UPDATE OF p`, authorID, postID).Scan(&audience); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return httpapi.NotFound("Публикация не найдена")
			}
			return err
		}
		if err := tx.QueryRow(ctx, `SELECT nextval(pg_get_serial_sequence('social_comments', 'id'))`).Scan(&comment.ID); err != nil {
			return err
		}
		var public *string
		var sealed []byte
		var keyID *string
		if audience == "public" {
			public = &body
		} else {
			value, err := s.content.SealString(body, commentAAD(comment.ID, authorID))
			if err != nil {
				return err
			}
			sealed, keyID = value.Ciphertext, &value.KeyID
		}
		if err := tx.QueryRow(ctx, `INSERT INTO social_comments (id, post_id, author_id, body_public, body_sealed, key_id)
		 OVERRIDING SYSTEM VALUE VALUES ($1, $2, $3, $4, $5, $6) RETURNING created_at`,
			comment.ID, postID, authorID, public, sealed, keyID).Scan(&comment.CreatedAt); err != nil {
			return err
		}
		return tx.QueryRow(ctx, `SELECT public_name FROM users WHERE id = $1`, authorID).Scan(&comment.AuthorName)
	})
	return comment, err
}

func commentAAD(commentID, authorID int64) string {
	return "social_comment:" + itoa(commentID) + ":author:" + itoa(authorID)
}
