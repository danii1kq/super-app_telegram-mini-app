package social

import (
	"context"
	"errors"
	"regexp"
	"strconv"
	"time"

	"github.com/danii1kq/srunal/internal/platform/httpapi"
	"github.com/jackc/pgx/v5"
)

// Person is another account as the viewer sees it: a name, a handle and whether
// the viewer already follows them.
type Person struct {
	ID         int64
	PublicName string
	Username   string
	Following  bool
	IsMe       bool
}

// People lists the accounts the viewer can start a conversation with, most
// recently active first. It is the directory the Mini App needs: without it the
// chat has no way to open a new dialog.
func (s *Service) People(ctx context.Context, viewerID int64, query string, limit int) ([]Person, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := s.pool.Query(ctx,
		`SELECT u.id, u.public_name, coalesce(u.username, ''),
		        EXISTS (SELECT 1 FROM social_follows f
		                 WHERE f.follower_id = $1 AND f.author_id = u.id) AS following
		   FROM users u
		  WHERE u.status = 'active'
		    AND u.id <> $1
		    AND NOT EXISTS (SELECT 1 FROM social_blocks b
		                     WHERE (b.blocker_id = $1 AND b.blocked_id = u.id)
		                        OR (b.blocker_id = u.id AND b.blocked_id = $1))
		    AND ($2 = '' OR u.public_name ILIKE '%' || $2 || '%'
		                  OR coalesce(u.username, '') ILIKE '%' || $2 || '%')
		  ORDER BY following DESC, u.public_name
		  LIMIT $3`, viewerID, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]Person, 0, limit)
	for rows.Next() {
		var person Person
		if err := rows.Scan(&person.ID, &person.PublicName, &person.Username, &person.Following); err != nil {
			return nil, err
		}
		out = append(out, person)
	}
	return out, rows.Err()
}

// Person returns one account by id, or a neutral not-found for a blocked or
// deleted one.
func (s *Service) Person(ctx context.Context, viewerID, userID int64) (Person, error) {
	var person Person
	err := s.pool.QueryRow(ctx,
		`SELECT u.id, u.public_name, coalesce(u.username, ''),
		        EXISTS (SELECT 1 FROM social_follows f
		                 WHERE f.follower_id = $1 AND f.author_id = u.id)
		   FROM users u
		  WHERE u.id = $2 AND u.status = 'active'
		    AND NOT EXISTS (SELECT 1 FROM social_blocks b
		                     WHERE (b.blocker_id = $1 AND b.blocked_id = u.id)
		                        OR (b.blocker_id = u.id AND b.blocked_id = $1))`,
		viewerID, userID,
	).Scan(&person.ID, &person.PublicName, &person.Username, &person.Following)
	if errors.Is(err, pgx.ErrNoRows) {
		return Person{}, httpapi.NotFound("Пользователь не найден")
	}
	if err != nil {
		return Person{}, err
	}
	person.IsMe = person.ID == viewerID
	return person, nil
}

type Profile struct {
	Person
	FollowsYou     bool
	Followers      int
	FollowingCount int
	PostsCount     int
	JoinedAt       time.Time
}

func (s *Service) Profile(ctx context.Context, viewerID, userID int64) (Profile, error) {
	person, err := s.Person(ctx, viewerID, userID)
	if err != nil {
		return Profile{}, err
	}
	profile := Profile{Person: person}
	err = s.pool.QueryRow(ctx, `SELECT u.created_at,
  EXISTS (SELECT 1 FROM social_follows f WHERE f.follower_id=$2 AND f.author_id=$1),
  (SELECT count(*) FROM social_follows f JOIN users follower ON follower.id=f.follower_id WHERE f.author_id=u.id AND follower.status='active'),
  (SELECT count(*) FROM social_follows f JOIN users followed ON followed.id=f.author_id WHERE f.follower_id=u.id AND followed.status='active'),
  (SELECT count(*) FROM social_posts p WHERE `+visiblePostSQL+` AND p.author_id=u.id AND p.community_id IS NULL)
  FROM users u WHERE u.id=$2`, viewerID, userID).Scan(&profile.JoinedAt, &profile.FollowsYou, &profile.Followers, &profile.FollowingCount, &profile.PostsCount)
	return profile, err
}

var telegramUsername = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]{4,31}$`)

func telegramURL(username string) *string {
	if !telegramUsername.MatchString(username) {
		return nil
	}
	value := "https://t.me/" + username
	return &value
}

type profileDTO struct {
	personDTO
	FollowsYou     bool    `json:"follows_you"`
	IsFriend       bool    `json:"is_friend"`
	Followers      int     `json:"followers"`
	FollowingCount int     `json:"following_count"`
	PostsCount     int     `json:"posts_count"`
	JoinedAt       string  `json:"joined_at"`
	TelegramURL    *string `json:"telegram_url"`
}

func toProfileDTO(profile Profile) profileDTO {
	return profileDTO{personDTO: toPersonDTO(profile.Person), FollowsYou: profile.FollowsYou,
		IsFriend: profile.Following && profile.FollowsYou, Followers: profile.Followers,
		FollowingCount: profile.FollowingCount, PostsCount: profile.PostsCount,
		JoinedAt: profile.JoinedAt.UTC().Format(time.RFC3339), TelegramURL: telegramURL(profile.Username)}
}

type personDTO struct {
	ID         string `json:"id"`
	PublicName string `json:"public_name"`
	Username   string `json:"username,omitempty"`
	Following  bool   `json:"following"`
	IsMe       bool   `json:"is_me"`
}

func toPersonDTO(person Person) personDTO {
	return personDTO{
		ID:         strconv.FormatInt(person.ID, 10),
		PublicName: person.PublicName,
		Username:   person.Username,
		Following:  person.Following,
		IsMe:       person.IsMe,
	}
}
