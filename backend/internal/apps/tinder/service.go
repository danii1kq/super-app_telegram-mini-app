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
	"github.com/danii1kq/srunal/internal/platform/media"
	"github.com/danii1kq/srunal/internal/platform/postgres"
	"github.com/danii1kq/srunal/internal/platform/secure"
)

type Service struct {
	pool    *postgres.Pool
	content *secure.Cipher
	// delivery seals outbox payloads. It is nil when notifications are not
	// configured, and every send path checks for that.
	delivery *secure.Cipher
}

func NewService(pool *postgres.Pool, content *secure.Cipher) *Service {
	return &Service{pool: pool, content: content}
}

// Profile is the viewer's own card.
type Profile struct {
	UserID     int64
	Bio        string
	PhotoURL   string
	University string
	Age        int
	Gender     string
	Seeking    []string
	Active     bool
}

// Card is one profile in the deck, as another player sees it.
type Card struct {
	UserID     int64
	Name       string
	Username   string
	Bio        string
	PhotoURL   string
	University string
	Age        int
	Gender     string
	LikesYou   bool
	HasPhoto   bool
	// PhotoVersion changes whenever the photo is replaced, so the client can
	// request a fresh image instead of a cached one.
	PhotoVersion string
	CreatedAt    time.Time
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

// genderValues is the accepted set. An empty string means "not specified" and
// is allowed so an existing profile stays valid.
var genderValues = map[string]bool{"": true, "male": true, "female": true, "other": true}

func validGender(value string) bool {
	return genderValues[value]
}

// normalizeSeeking validates the preference list and removes duplicates.
//
// An empty list is meaningful: it means "no preference" and shows everyone, so
// a player who does not care is never filtered out of anyone's deck.
func normalizeSeeking(values []string) ([]string, error) {
	out := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		if value == "" {
			continue
		}
		if !genderValues[value] {
			return nil, httpapi.Validation(map[string]string{"seeking": "Некорректный выбор"}, "Проверьте поля")
		}
		if seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out, nil
}

func photoAAD(userID int64) string {
	return "tinder_photo:" + itoa(userID)
}

// maxPhotoBytes bounds a profile photo. A phone photo is well under this.
const maxPhotoBytes = 5 << 20

// SavePhoto normalizes and seals the profile photo.
//
// The bytes are sniffed, never trusted from the request header: a caller can
// claim any content type. The stored image is re-encoded by media.Normalize, so
// an oversized or exotic file cannot reach the database as-is.
func (s *Service) SavePhoto(ctx context.Context, userID int64, raw []byte) (string, error) {
	if len(raw) == 0 {
		return "", httpapi.Validation(map[string]string{"photo": "Файл не выбран"}, "Проверьте поля")
	}
	if len(raw) > maxPhotoBytes {
		return "", httpapi.Validation(map[string]string{"photo": "Файл больше 5 МБ"}, "Проверьте поля")
	}
	if media.DetectMime(raw) == "" {
		return "", httpapi.Validation(map[string]string{"photo": "Это не изображение"}, "Проверьте поля")
	}
	normalized, err := media.Normalize(raw)
	if err != nil {
		return "", httpapi.Validation(map[string]string{"photo": "Не удалось обработать изображение"}, "Проверьте поля")
	}
	sealed, err := s.content.Seal(normalized.Data, photoAAD(userID))
	if err != nil {
		return "", err
	}
	result, err := s.pool.Exec(ctx,
		`UPDATE tinder_profiles
		    SET photo_sealed = $2, photo_key_id = $3, photo_mime = $4, updated_at = now()
		  WHERE user_id = $1`,
		userID, sealed.Ciphertext, sealed.KeyID, normalized.Mime)
	if err != nil {
		return "", err
	}
	if result.RowsAffected() == 0 {
		return "", httpapi.NotFound("Сначала создайте анкету")
	}
	return normalized.Mime, nil
}

// Photo returns the decrypted photo for a profile the viewer is allowed to see.
//
// Visibility follows the deck rules: the owner always sees their own, and
// anyone else must not be blocked in either direction. The photo is not public.
func (s *Service) Photo(ctx context.Context, viewerID, ownerID int64) ([]byte, string, error) {
	if viewerID != ownerID {
		var allowed bool
		if err := s.pool.QueryRow(ctx,
			`SELECT EXISTS (SELECT 1 FROM tinder_profiles p JOIN users u ON u.id=p.user_id WHERE p.user_id = $1 AND p.active AND u.status='active')
			    AND NOT EXISTS (SELECT 1 FROM social_blocks b
			                     WHERE (b.blocker_id = $1 AND b.blocked_id = $2)
			                        OR (b.blocker_id = $2 AND b.blocked_id = $1))`,
			ownerID, viewerID,
		).Scan(&allowed); err != nil {
			return nil, "", err
		}
		if !allowed {
			return nil, "", httpapi.NotFound("Фото не найдено")
		}
	}

	var sealed []byte
	var keyID, mime string
	err := s.pool.QueryRow(ctx,
		`SELECT photo_sealed, photo_key_id, photo_mime FROM tinder_profiles WHERE user_id = $1`,
		ownerID,
	).Scan(&sealed, &keyID, &mime)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && keyID == "") {
		return nil, "", httpapi.NotFound("Фото не найдено")
	}
	if err != nil {
		return nil, "", err
	}
	data, err := s.content.Open(secure.Sealed{KeyID: keyID, Ciphertext: sealed}, photoAAD(ownerID))
	if err != nil {
		return nil, "", err
	}
	return data, mime, nil
}

// HasPhoto reports whether a profile has a stored photo, for the card flag.
func (s *Service) HasPhoto(ctx context.Context, userID int64) (bool, error) {
	var has bool
	err := s.pool.QueryRow(ctx,
		`SELECT coalesce(photo_key_id <> '', false) FROM tinder_profiles WHERE user_id = $1`,
		userID).Scan(&has)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	return has, err
}

// SaveProfile creates or replaces the viewer's card. Bio is sealed like every
// other free-form text in this project.
func (s *Service) SaveProfile(ctx context.Context, userID int64, bio, photoURL, university, gender string, seeking []string, age int, active bool) (Profile, error) {
	if len([]rune(bio)) > 500 {
		return Profile{}, httpapi.Validation(map[string]string{"bio": "Слишком длинный текст"}, "Проверьте поля")
	}
	if age != 0 && (age < 18 || age > 120) {
		return Profile{}, httpapi.Validation(map[string]string{"age": "Возраст от 18 до 120"}, "Проверьте поля")
	}
	if len(photoURL) > 500 || len(university) > 80 {
		return Profile{}, httpapi.Validation(map[string]string{"photo_url": "Слишком длинное значение"}, "Проверьте поля")
	}
	if !validGender(gender) {
		return Profile{}, httpapi.Validation(map[string]string{"gender": "Выберите пол"}, "Проверьте поля")
	}
	seeking, err := normalizeSeeking(seeking)
	if err != nil {
		return Profile{}, err
	}

	sealed, err := s.content.SealString(bio, profileAAD(userID))
	if err != nil {
		return Profile{}, err
	}
	_, err = s.pool.Exec(ctx,
		`INSERT INTO tinder_profiles (user_id, bio_sealed, key_id, photo_url, university, gender, seeking, age, active)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		 ON CONFLICT (user_id) DO UPDATE
		    SET bio_sealed = EXCLUDED.bio_sealed,
		        key_id = EXCLUDED.key_id,
		        photo_url = EXCLUDED.photo_url,
		        university = EXCLUDED.university,
		        gender = EXCLUDED.gender,
		        seeking = EXCLUDED.seeking,
		        age = EXCLUDED.age,
		        active = EXCLUDED.active,
		        updated_at = now()`,
		userID, sealed.Ciphertext, sealed.KeyID, photoURL, university, gender, seeking, age, active)
	if err != nil {
		return Profile{}, err
	}
	return Profile{UserID: userID, Bio: bio, PhotoURL: photoURL, University: university, Gender: gender, Seeking: seeking, Age: age, Active: active}, nil
}

// OwnProfile returns the viewer's card, or a zero profile when none exists.
func (s *Service) OwnProfile(ctx context.Context, userID int64) (Profile, bool, error) {
	var sealed []byte
	var keyID string
	var profile Profile
	err := s.pool.QueryRow(ctx,
		`SELECT bio_sealed, key_id, photo_url, university, gender, seeking, coalesce(age, 0), active
		   FROM tinder_profiles WHERE user_id = $1`, userID,
	).Scan(&sealed, &keyID, &profile.PhotoURL, &profile.University, &profile.Gender, &profile.Seeking, &profile.Age, &profile.Active)
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
//
// The gender filter runs both ways, which is the only correct reading of a
// preference: a card appears only when the viewer is looking for that gender
// AND that person is looking for the viewer's gender. An empty seeking list
// means no preference, so a player who did not choose is never hidden.
func (s *Service) Deck(ctx context.Context, viewerID int64, limit int) ([]Card, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	rows, err := s.pool.Query(ctx,
		`WITH me AS (
		     SELECT gender, seeking FROM tinder_profiles WHERE user_id = $1
		 )
		 SELECT p.user_id, u.public_name, coalesce(u.username, ''),
		        p.bio_sealed, p.key_id, p.photo_url, p.university, coalesce(p.age, 0), p.gender,
		        coalesce(p.photo_key_id <> '', false),
		        to_char(p.updated_at, 'YYYYMMDDHH24MISSUS'),
		        EXISTS (SELECT 1 FROM tinder_likes l WHERE l.from_id = p.user_id AND l.to_id = $1) AS likes_you,
		        p.updated_at
		   FROM tinder_profiles p
		   JOIN users u ON u.id = p.user_id
		   LEFT JOIN me ON true
		  WHERE p.active
		    AND p.user_id <> $1
		    AND u.status = 'active'
		    AND NOT EXISTS (SELECT 1 FROM tinder_likes l WHERE l.from_id = $1 AND l.to_id = p.user_id)
		    AND NOT EXISTS (SELECT 1 FROM tinder_passes x WHERE x.from_id = $1 AND x.to_id = p.user_id)
		    AND NOT EXISTS (SELECT 1 FROM social_blocks b
		                     WHERE (b.blocker_id = $1 AND b.blocked_id = p.user_id)
		                        OR (b.blocker_id = p.user_id AND b.blocked_id = $1))
		    -- The viewer wants this gender (or has no preference).
		    AND (coalesce(cardinality(me.seeking), 0) = 0 OR p.gender = ANY(me.seeking))
		    -- This person wants the viewer's gender (or has no preference).
		    AND (coalesce(cardinality(p.seeking), 0) = 0 OR me.gender = ANY(p.seeking))
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
			&card.PhotoURL, &card.University, &card.Age, &card.Gender, &card.HasPhoto, &card.PhotoVersion, &card.LikesYou, &card.CreatedAt); err != nil {
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
		ids := []int64{min(fromID, toID), max(fromID, toID)}
		rows, err := tx.Query(ctx, `SELECT id FROM users WHERE id=ANY($1::bigint[]) AND status='active' ORDER BY id FOR UPDATE`, ids)
		if err != nil {
			return err
		}
		count := 0
		for rows.Next() {
			count++
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return err
		}
		if count != 2 {
			return httpapi.NotFound("Анкета не найдена")
		}
		var exists bool
		if err := tx.QueryRow(ctx,
			`SELECT (SELECT count(*) FROM tinder_profiles WHERE user_id=ANY($1::bigint[]) AND active)=2
			 AND NOT EXISTS(SELECT 1 FROM social_blocks WHERE (blocker_id=$2 AND blocked_id=$3) OR (blocker_id=$3 AND blocked_id=$2))`, ids, fromID, toID,
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
			if s.delivery != nil {
				if err := s.notifyLike(ctx, tx, fromID, toID); err != nil {
					return err
				}
			}
			return nil
		}

		a, b := fromID, toID
		if a > b {
			a, b = b, a
		}
		created := true
		err = tx.QueryRow(ctx, `INSERT INTO tinder_matches(user_a,user_b)VALUES($1,$2)ON CONFLICT(user_a,user_b)DO NOTHING RETURNING id`, a, b).Scan(&matchID)
		if errors.Is(err, pgx.ErrNoRows) {
			created = false
			err = tx.QueryRow(ctx, `SELECT id FROM tinder_matches WHERE user_a=$1 AND user_b=$2`, a, b).Scan(&matchID)
		}
		if err != nil {
			return err
		}
		matched = true
		// Both sides hear about a match; the like that caused it is not
		// announced separately, because the match already says it.
		if s.delivery != nil && created {
			if err := s.notifyMatch(ctx, tx, a, b); err != nil {
				return err
			}
		}
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
		  WHERE (m.user_a = $1 OR m.user_b = $1) AND u.status='active'
		    AND NOT EXISTS(SELECT 1 FROM social_blocks b WHERE (b.blocker_id=$1 AND b.blocked_id=u.id) OR (b.blocker_id=u.id AND b.blocked_id=$1))
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
