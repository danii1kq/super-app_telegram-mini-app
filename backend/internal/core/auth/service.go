package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/danii1kq/srunal/internal/platform/httpapi"
	"github.com/danii1kq/srunal/internal/platform/postgres"
	"github.com/danii1kq/srunal/internal/platform/secure"
)

// User is the account as the rest of the server sees it.
type User struct {
	ID         int64
	PublicName string
	Username   string
	Status     string
}

// Service owns accounts and sessions.
type Service struct {
	pool    *postgres.Pool
	index   *secure.LookupIndex
	content *secure.Cipher
	ttl     time.Duration
}

func NewService(pool *postgres.Pool, index *secure.LookupIndex, content *secure.Cipher, ttl time.Duration) *Service {
	return &Service{pool: pool, index: index, content: content, ttl: ttl}
}

// Login finds or creates the account for a verified payload and issues a
// session token. The returned token is the only copy: the database holds its
// hash.
func (s *Service) Login(ctx context.Context, data InitData) (User, string, error) {
	lookup := s.index.Of(fmt.Sprintf("%d", data.TelegramID))
	publicName := data.FirstName
	if publicName == "" {
		publicName = "Пользователь"
	}

	var user User
	err := s.pool.InTx(ctx, func(tx pgx.Tx) error {
		var id int64
		// Upsert, not select-then-insert: React StrictMode fires the login twice
		// in development, and two concurrent inserts of the same account raced
		// into a unique violation. ON CONFLICT makes the second one an update.
		sealed, err := s.content.SealString(fmt.Sprintf("%d", data.TelegramID), "user:telegram_id")
		if err != nil {
			return err
		}
		err = tx.QueryRow(ctx,
			`INSERT INTO users (telegram_id_lookup, telegram_id_sealed, key_id, public_name, username)
			 VALUES ($1, $2, $3, $4, nullif($5, ''))
			 ON CONFLICT (telegram_id_lookup) DO UPDATE
			    SET public_name = excluded.public_name,
			        username = excluded.username,
			        updated_at = now()
			 RETURNING id, public_name, coalesce(username, ''), status`,
			lookup, sealed.Ciphertext, sealed.KeyID, publicName, data.Username,
		).Scan(&id, &user.PublicName, &user.Username, &user.Status)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO user_settings (user_id) VALUES ($1) ON CONFLICT DO NOTHING`, id,
		); err != nil {
			return err
		}

		user.ID = id
		return nil
	})
	if err != nil {
		return User{}, "", err
	}

	if user.Status != "active" {
		return User{}, "", httpapi.Forbidden("Аккаунт заблокирован")
	}

	token, err := secure.NewToken(32)
	if err != nil {
		return User{}, "", err
	}

	// The verified payload is kept with the session (sealed) so the trusted hop
	// to SruDoc and SruDay can present it while the session is alive.
	sealed, err := s.content.SealString(data.Raw, sessionAAD(user.ID))
	if err != nil {
		return User{}, "", err
	}

	_, err = s.pool.Exec(ctx,
		`INSERT INTO sessions (token_hash, user_id, expires_at, init_data_sealed, init_data_key_id)
		 VALUES ($1, $2, $3, $4, $5)`,
		secure.TokenHash(token), user.ID, time.Now().Add(s.ttl), sealed.Ciphertext, sealed.KeyID,
	)
	if err != nil {
		return User{}, "", err
	}
	return user, token, nil
}

// InitDataFor returns the verified Telegram payload stored with the session.
func (s *Service) InitDataFor(ctx context.Context, userID int64, token string) (string, error) {
	var sealed []byte
	var keyID string
	err := s.pool.QueryRow(ctx,
		`SELECT init_data_sealed, init_data_key_id FROM sessions
		  WHERE token_hash = $1 AND user_id = $2 AND revoked_at IS NULL AND expires_at > now()`,
		secure.TokenHash(token), userID,
	).Scan(&sealed, &keyID)
	if err != nil {
		return "", nil
	}
	return s.content.OpenString(secure.Sealed{KeyID: keyID, Ciphertext: sealed}, sessionAAD(userID))
}

func sessionAAD(userID int64) string {
	return "session_init_data:user:" + itoa(userID)
}

func itoa(value int64) string {
	if value == 0 {
		return "0"
	}
	var buffer [20]byte
	position := len(buffer)
	for value > 0 {
		position--
		buffer[position] = byte('0' + value%10)
		value /= 10
	}
	return string(buffer[position:])
}

// Authenticate resolves a bearer token to its user. Every request goes through
// here: no handler trusts an id from the body.
func (s *Service) Authenticate(ctx context.Context, token string) (User, error) {
	if token == "" {
		return User{}, httpapi.Unauthorized("Требуется вход")
	}

	var user User
	err := s.pool.QueryRow(ctx,
		`SELECT u.id, u.public_name, coalesce(u.username, ''), u.status
		   FROM sessions s
		   JOIN users u ON u.id = s.user_id
		  WHERE s.token_hash = $1
		    AND s.revoked_at IS NULL
		    AND s.expires_at > now()`, secure.TokenHash(token),
	).Scan(&user.ID, &user.PublicName, &user.Username, &user.Status)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, httpapi.Unauthorized("Сессия недействительна")
	}
	if err != nil {
		return User{}, err
	}
	if user.Status != "active" {
		return User{}, httpapi.Forbidden("Аккаунт заблокирован")
	}
	return user, nil
}

// Logout revokes the session.
func (s *Service) Logout(ctx context.Context, token string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE sessions SET revoked_at = now() WHERE token_hash = $1 AND revoked_at IS NULL`,
		secure.TokenHash(token),
	)
	return err
}
