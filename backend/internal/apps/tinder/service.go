// Package tinder owns SruTinder: a profile, a deck, likes and matches.
//
// The model is deliberately the plain one: a profile, a like, a mutual like
// becomes a match, and a match opens a chat. There is no scheduling layer and
// no time filter; the deck is ordered so that people who already liked the
// viewer come first, because that is the only ordering that reliably produces
// a match in a small network.
package tinder

import (
	"context"
	"errors"
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

// Profile is the viewer's own card.
type Profile struct {
	UserID   int64
	Bio      string
	PhotoURL string
	City     string
	Age      int
	Active   bool
}

// Card is one profile in the deck, as another player sees it.
type Card struct {
	UserID    int64
	Name      string
	Username  string
	Bio       string
	PhotoURL  string
	City      string
	Age       int
	LikesYou  bool
	CreatedAt time.Time
}

// Match is a mutual like.
type Match struct {
	ID        int64
	PartnerID int64
	Name      string
	Username  string
	PhotoURL  string
	CreatedAt time.Time
}

func profileAAD(userID int64) string {
	return "tinder_profile:" + itoa(userID)
}

// SaveProfile creates or replaces the viewer's card. Bio is sealed like every
// other free-form text in this project.
func (s *Service) SaveProfile(ctx context.Context, userID int64, bio, photoURL, city string, age int, active bool) (Profile, error) {
	if len([]rune(bio)) > 500 {
		return Profile{}, httpapi.Validation(map[string]string{"bio": "Слишком длинный текст"}, "Проверьте поля")
	}
	if age != 0 && (age < 18 || age > 120) {
		return Profile{}, httpapi.Validation(map[string]string{"age": "Возраст от 18 до 120"}, "Проверьте поля")
	}
	if len(photoURL) > 500 || len(city) > 80 {
		return Profile{}, httpapi.Validation(map[string]string{"photo_url": "Слишком длинное значение"}, "Проверьте поля")
	}

	sealed, err := s.content.SealString(bio, profileAAD(userID))
	if err != nil {
		return Profile{}, err
	}
	_, err = s.pool.Exec(ctx,
		`INSERT INTO tinder_profiles (user_id, bio_sealed, key_id, photo_url, city, age, active)
		 VALUES ($1,$2,$3,$4,$5,$6,$7)
		 ON CONFLICT (user_id) DO UPDATE
		    SET bio_sealed = EXCLUDED.bio_sealed,
		        key_id = EXCLUDED.key_id,
		        photo_url = EXCLUDED.photo_url,
		        city = EXCLUDED.city,
		        age = EXCLUDED.age,
		        active = EXCLUDED.active,
		        updated_at = now()`,
		userID, sealed.Ciphertext, sealed.KeyID, photoURL, city, age, active)
	if err != nil {
		return Profile{}, err
	}
	return Profile{UserID: userID, Bio: bio, PhotoURL: photoURL, City: city, Age: age, Active: active}, nil
}

// OwnProfile returns the viewer's card, or a zero profile when none exists.
func (s *Service) OwnProfile(ctx context.Context, userID int64) (Profile, bool, error) {
	var sealed []byte
	var keyID string
	var profile Profile
	err := s.pool.QueryRow(ctx,
		`SELECT bio_sealed, key_id, photo_url, city, coalesce(age, 0), active
		   FROM tinder_profiles WHERE user_id = $1`, userID,
	).Scan(&sealed, &keyID, &profile.PhotoURL, &profile.City, &profile.Age, &profile.Active)
	if errors.Is(err, pgx.ErrNoRows) {
		return Profile{}, false, nil
	}
	if err != nil {
		return Profile{}, false, err
	}
	bio, err := s.content.OpenString(secure.Sealed{KeyID: keyID, Ciphertext: sealed}, profileAAD(userID))
	if err != nil {
		return Profile{}, false, err
	}
	profile.UserID = userID
	profile.Bio = bio
	return profile, true, nil
}

// Deck returns the next cards for the viewer.
//
// Excluded: the viewer, anyone already liked or passed, anyone blocked in
// either direction, and profiles that are switched off. Ordered so that a
// profile which already liked the viewer comes first: in a small network that
// is what turns a session into a match instead of a pile of maybes.
func (s *Service) Deck(ctx context.Context, viewerID int64, limit int) ([]Card, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	rows, err := s.pool.Query(ctx,
		`SELECT p.user_id, u.public_name, coalesce(u.username, ''),
		        p.bio_sealed, p.key_id, p.photo_url, p.city, coalesce(p.age, 0),
		        EXISTS (SELECT 1 FROM tinder_likes l WHERE l.from_id = p.user_id AND l.to_id = $1) AS likes_you,
		        p.updated_at
		   FROM tinder_profiles p
		   JOIN users u ON u.id = p.user_id
		  WHERE p.active
		    AND p.user_id <> $1
		    AND u.status = 'active'
		    AND NOT EXISTS (SELECT 1 FROM tinder_likes l WHERE l.from_id = $1 AND l.to_id = p.user_id)
		    AND NOT EXISTS (SELECT 1 FROM tinder_passes x WHERE x.from_id = $1 AND x.to_id = p.user_id)
		    AND NOT EXISTS (SELECT 1 FROM social_blocks b
		                     WHERE (b.blocker_id = $1 AND b.blocked_id = p.user_id)
		                        OR (b.blocker_id = p.user_id AND b.blocked_id = $1))
		  ORDER BY likes_you DESC, p.updated_at DESC
		  LIMIT $2`, viewerID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]Card, 0, limit)
	for rows.Next() {
		var card Card
		var sealed []byte
		var keyID string
		if err := rows.Scan(&card.UserID, &card.Name, &card.Username, &sealed, &keyID,
			&card.PhotoURL, &card.City, &card.Age, &card.LikesYou, &card.CreatedAt); err != nil {
			return nil, err
		}
		bio, err := s.content.OpenString(secure.Sealed{KeyID: keyID, Ciphertext: sealed}, profileAAD(card.UserID))
		if err != nil {
			return nil, err
		}
		card.Bio = bio
		out = append(out, card)
	}
	return out, rows.Err()
}

// Like records a like and reports whether it produced a match.
//
// The mutual check and the match insert happen in one transaction with the
// pair locked in a stable order, so two simultaneous likes cannot create two
// matches or miss one.
func (s *Service) Like(ctx context.Context, fromID, toID int64) (matched bool, matchID int64, err error) {
	if fromID == toID {
		return false, 0, httpapi.BadRequest("own_profile", "Нельзя лайкнуть себя")
	}
	err = s.pool.InTx(ctx, func(tx pgx.Tx) error {
		var exists bool
		if err := tx.QueryRow(ctx,
			`SELECT EXISTS (SELECT 1 FROM tinder_profiles WHERE user_id = $1 AND active)`, toID,
		).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			return httpapi.NotFound("Анкета не найдена")
		}

		if _, err := tx.Exec(ctx,
			`INSERT INTO tinder_likes (from_id, to_id) VALUES ($1,$2) ON CONFLICT DO NOTHING`,
			fromID, toID); err != nil {
			return err
		}

		var mutual bool
		if err := tx.QueryRow(ctx,
			`SELECT EXISTS (SELECT 1 FROM tinder_likes WHERE from_id = $1 AND to_id = $2)`,
			toID, fromID,
		).Scan(&mutual); err != nil {
			return err
		}
		if !mutual {
			return nil
		}

		a, b := fromID, toID
		if a > b {
			a, b = b, a
		}
		if err := tx.QueryRow(ctx,
			`INSERT INTO tinder_matches (user_a, user_b) VALUES ($1,$2)
			 ON CONFLICT (user_a, user_b) DO UPDATE SET user_a = EXCLUDED.user_a
			 RETURNING id`, a, b,
		).Scan(&matchID); err != nil {
			return err
		}
		matched = true
		return nil
	})
	if err != nil {
		return false, 0, err
	}
	return matched, matchID, nil
}

// Pass records a "no" so the card does not come back.
func (s *Service) Pass(ctx context.Context, fromID, toID int64) error {
	if fromID == toID {
		return httpapi.BadRequest("own_profile", "Нельзя пропустить себя")
	}
	_, err := s.pool.Exec(ctx,
		`INSERT INTO tinder_passes (from_id, to_id) VALUES ($1,$2) ON CONFLICT DO NOTHING`,
		fromID, toID)
	return err
}

// Matches lists the viewer's matches, newest first.
func (s *Service) Matches(ctx context.Context, viewerID int64) ([]Match, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT m.id,
		        CASE WHEN m.user_a = $1 THEN m.user_b ELSE m.user_a END AS partner,
		        u.public_name, coalesce(u.username, ''), coalesce(p.photo_url, ''), m.created_at
		   FROM tinder_matches m
		   JOIN users u ON u.id = CASE WHEN m.user_a = $1 THEN m.user_b ELSE m.user_a END
		   LEFT JOIN tinder_profiles p ON p.user_id = u.id
		  WHERE m.user_a = $1 OR m.user_b = $1
		  ORDER BY m.created_at DESC
		  LIMIT 100`, viewerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]Match, 0, 8)
	for rows.Next() {
		var item Match
		if err := rows.Scan(&item.ID, &item.PartnerID, &item.Name, &item.Username, &item.PhotoURL, &item.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

// LikesMe counts incoming likes, for the badge on the deck.
func (s *Service) LikesMe(ctx context.Context, viewerID int64) (int, error) {
	var count int
	err := s.pool.QueryRow(ctx,
		`SELECT count(*) FROM tinder_likes l
		  WHERE l.to_id = $1
		    AND NOT EXISTS (SELECT 1 FROM tinder_likes x WHERE x.from_id = $1 AND x.to_id = l.from_id)
		    AND NOT EXISTS (SELECT 1 FROM social_blocks b
		                     WHERE (b.blocker_id = $1 AND b.blocked_id = l.from_id)
		                        OR (b.blocker_id = l.from_id AND b.blocked_id = $1))`,
		viewerID).Scan(&count)
	return count, err
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
