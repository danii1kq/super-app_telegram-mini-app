package social

import (
	"context"
	"strconv"

	"github.com/danii1kq/srunal/internal/platform/httpapi"
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
	if err != nil {
		return Person{}, httpapi.NotFound("Пользователь не найден")
	}
	person.IsMe = person.ID == viewerID
	return person, nil
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
