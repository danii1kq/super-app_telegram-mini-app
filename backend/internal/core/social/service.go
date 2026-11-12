// Package social owns the SruNal feed: posts, comments, reactions, follows and
// communities. It never touches chat or app tables.
package social

import (
	"context"
	"errors"
	"strings"
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

type Post struct {
	ID          int64
	AuthorID    int64
	AuthorName  string
	Body        string
	Audience    string
	CommunityID *int64
	CreatedAt   time.Time
	Likes       int
	Comments    int
	LikedByMe   bool
	Views       int
}

// Feed returns posts visible to viewer: own posts, followed authors and the
// communities the viewer belongs to. The restriction lives in the SQL, not in
// the handler.
func (s *Service) Feed(ctx context.Context, viewerID int64, beforeID int64, limit int) ([]Post, error) {
	return s.posts(ctx, viewerID, 0, beforeID, limit)
}

func (s *Service) posts(ctx context.Context, viewerID, authorID, beforeID int64, limit int) ([]Post, error) {
	if limit <= 0 || limit > 51 {
		limit = 20
	}

	rows, err := s.pool.Query(ctx,
		`SELECT p.id, p.author_id, u.public_name,
		        coalesce(p.body_public, ''), p.body_sealed, p.key_id,
		        p.audience, p.community_id, p.created_at,
		        (SELECT count(*) FROM social_reactions r WHERE r.post_id = p.id) AS likes,
		        (SELECT count(*) FROM social_comments c WHERE c.post_id = p.id AND c.deleted_at IS NULL) AS comments,
		        EXISTS (SELECT 1 FROM social_reactions r2 WHERE r2.post_id = p.id AND r2.user_id = $1) AS liked,
		        (SELECT count(*) FROM social_post_views v WHERE v.post_id = p.id) AS views
		   FROM social_posts p
		   JOIN users u ON u.id = p.author_id
		  WHERE `+visiblePostSQL+`
		    AND ($2::bigint = 0 OR p.id < $2)
		    AND ($4::bigint = 0 OR (p.author_id = $4 AND p.community_id IS NULL))
		  ORDER BY p.id DESC
		  LIMIT $3`, viewerID, beforeID, limit, authorID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	posts := make([]Post, 0, limit)
	for rows.Next() {
		var post Post
		var sealed []byte
		var keyID *string
		if err := rows.Scan(&post.ID, &post.AuthorID, &post.AuthorName,
			&post.Body, &sealed, &keyID, &post.Audience, &post.CommunityID,
			&post.CreatedAt, &post.Likes, &post.Comments, &post.LikedByMe, &post.Views,
		); err != nil {
			return nil, err
		}
		if len(sealed) > 0 && keyID != nil {
			body, err := s.content.OpenString(secure.Sealed{KeyID: *keyID, Ciphertext: sealed}, postAAD(post.ID, post.AuthorID))
			if err != nil {
				// A broken ciphertext is an error, never an empty post.
				return nil, err
			}
			post.Body = body
		}
		posts = append(posts, post)
	}
	return posts, rows.Err()
}

type NewPost struct {
	Body      string
	Audience  string
	Community *int64
}

func (s *Service) CreatePost(ctx context.Context, authorID int64, input NewPost) (Post, error) {
	input.Body = strings.TrimSpace(input.Body)
	if input.Body == "" {
		return Post{}, httpapi.Validation(map[string]string{"body": "Текст не может быть пустым"}, "Проверьте поля")
	}
	if len([]rune(input.Body)) > 4000 {
		return Post{}, httpapi.Validation(map[string]string{"body": "Слишком длинный текст"}, "Проверьте поля")
	}
	if input.Audience != "public" && input.Audience != "followers" && input.Audience != "community" {
		return Post{}, httpapi.BadRequest("invalid_audience", "Выберите аудиторию публикации")
	}
	if (input.Audience == "community") != (input.Community != nil) || (input.Community != nil && *input.Community <= 0) {
		return Post{}, httpapi.BadRequest("invalid_community", "Проверьте сообщество и аудиторию")
	}

	var id int64
	err := s.pool.InTx(ctx, func(tx pgx.Tx) error {
		if input.Community != nil {
			// Posting into a community requires active membership.
			var member bool
			if err := tx.QueryRow(ctx,
				`SELECT EXISTS (SELECT 1 FROM social_members m JOIN social_communities c ON c.id = m.community_id
				                 WHERE m.community_id = $1 AND m.user_id = $2 AND m.state = 'active' AND c.state = 'active')`,
				*input.Community, authorID,
			).Scan(&member); err != nil {
				return err
			}
			if !member {
				return httpapi.Forbidden("Нет доступа к сообществу")
			}
		}

		if input.Audience == "public" {
			return tx.QueryRow(ctx,
				`INSERT INTO social_posts (author_id, community_id, audience, body_public)
				 VALUES ($1, $2, 'public', $3) RETURNING id`,
				authorID, input.Community, input.Body,
			).Scan(&id)
		}

		if err := tx.QueryRow(ctx,
			`SELECT nextval(pg_get_serial_sequence('social_posts', 'id'))`,
		).Scan(&id); err != nil {
			return err
		}
		sealed, err := s.content.SealString(input.Body, postAAD(id, authorID))
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx,
			`INSERT INTO social_posts (id, author_id, community_id, audience, body_sealed, key_id)
			 OVERRIDING SYSTEM VALUE VALUES ($1, $2, $3, $4, $5, $6)`,
			id, authorID, input.Community, input.Audience, sealed.Ciphertext, sealed.KeyID,
		)
		return err
	})
	if err != nil {
		return Post{}, err
	}
	return s.byID(ctx, id, authorID)
}

func (s *Service) byID(ctx context.Context, id, viewerID int64) (Post, error) {
	var post Post
	var sealed []byte
	var keyID *string
	err := s.pool.QueryRow(ctx,
		`SELECT p.id, p.author_id, u.public_name, coalesce(p.body_public, ''),
		        p.body_sealed, p.key_id, p.audience, p.community_id, p.created_at,
		        (SELECT count(*) FROM social_reactions r WHERE r.post_id = p.id),
		        (SELECT count(*) FROM social_comments c WHERE c.post_id = p.id AND c.deleted_at IS NULL),
		        EXISTS (SELECT 1 FROM social_reactions r WHERE r.post_id = p.id AND r.user_id = $1),
		        (SELECT count(*) FROM social_post_views v WHERE v.post_id = p.id)
		   FROM social_posts p JOIN users u ON u.id = p.author_id
		  WHERE `+visiblePostSQL+` AND p.id = $2`, viewerID, id,
	).Scan(&post.ID, &post.AuthorID, &post.AuthorName, &post.Body, &sealed, &keyID,
		&post.Audience, &post.CommunityID, &post.CreatedAt, &post.Likes, &post.Comments, &post.LikedByMe, &post.Views)
	if errors.Is(err, pgx.ErrNoRows) {
		return Post{}, httpapi.NotFound("Публикация не найдена")
	}
	if err != nil {
		return Post{}, err
	}
	if len(sealed) > 0 && keyID != nil {
		body, err := s.content.OpenString(secure.Sealed{KeyID: *keyID, Ciphertext: sealed}, postAAD(post.ID, post.AuthorID))
		if err != nil {
			return Post{}, err
		}
		post.Body = body
	}
	return post, nil
}

// DeletePost soft-deletes the author's own post.
func (s *Service) DeletePost(ctx context.Context, authorID, postID int64) error {
	tag, err := s.pool.Exec(ctx,
		`UPDATE social_posts SET deleted_at = now()
		  WHERE id = $1 AND author_id = $2 AND deleted_at IS NULL`, postID, authorID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return httpapi.NotFound("Публикация не найдена")
	}
	return nil
}

// ToggleReaction adds or removes the viewer's reaction.
func (s *Service) ToggleReaction(ctx context.Context, userID, postID int64, kind string) (bool, error) {
	if kind != "like" && kind != "heart" && kind != "fire" {
		kind = "like"
	}
	var added bool
	err := s.pool.InTx(ctx, func(tx pgx.Tx) error {
		var visibleID int64
		if err := tx.QueryRow(ctx, `SELECT p.id FROM social_posts p WHERE `+visiblePostSQL+` AND p.id = $2 FOR UPDATE OF p`, userID, postID).Scan(&visibleID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return httpapi.NotFound("Публикация не найдена")
			}
			return err
		}
		tag, err := tx.Exec(ctx,
			`DELETE FROM social_reactions WHERE post_id = $1 AND user_id = $2 AND kind = $3`,
			postID, userID, kind)
		if err != nil {
			return err
		}
		if tag.RowsAffected() > 0 {
			added = false
			return nil
		}
		_, err = tx.Exec(ctx,
			`INSERT INTO social_reactions (post_id, user_id, kind) VALUES ($1, $2, $3)`,
			postID, userID, kind)
		added = true
		return err
	})
	return added, err
}

// Follow subscribes the viewer to an author.
func (s *Service) Follow(ctx context.Context, followerID, authorID int64) error {
	if followerID == authorID {
		return httpapi.BadRequest("self_follow", "Нельзя подписаться на себя")
	}
	tag, err := s.pool.Exec(ctx,
		`INSERT INTO social_follows (follower_id, author_id)
		 SELECT $1, u.id FROM users u WHERE u.id = $2 AND u.status = 'active'
		 AND NOT EXISTS (SELECT 1 FROM social_blocks b
		  WHERE (b.blocker_id = $1 AND b.blocked_id = u.id)
		     OR (b.blocker_id = u.id AND b.blocked_id = $1))
		 ON CONFLICT (follower_id, author_id) DO UPDATE SET author_id = excluded.author_id`, followerID, authorID)
	if err == nil && tag.RowsAffected() == 0 {
		return httpapi.NotFound("Пользователь не найден")
	}
	return err
}

func (s *Service) Unfollow(ctx context.Context, followerID, authorID int64) error {
	_, err := s.pool.Exec(ctx,
		`DELETE FROM social_follows WHERE follower_id = $1 AND author_id = $2`, followerID, authorID)
	return err
}

// postAAD binds a ciphertext to its post and author.
func postAAD(postID, authorID int64) string {
	return "social_post:" + itoa(postID) + ":author:" + itoa(authorID)
}

func itoa(value int64) string {
	if value == 0 {
		return "0"
	}
	negative := value < 0
	if negative {
		value = -value
	}
	var buffer [20]byte
	position := len(buffer)
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
