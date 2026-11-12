// Package pulse owns SruPulse: short-lived "I am free until ..." statuses.
//
// Visibility is decided by the audience rows, the block list and the server
// clock. An expired or closed status accepts no new responses, even through an
// old link, because the check happens in the transaction on every write.
package pulse

import (
	"context"
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

type Status struct {
	ID         int64
	AuthorID   int64
	AuthorName string
	Text       string
	ExpiresAt  time.Time
	ClosedAt   *time.Time
	CreatedAt  time.Time
	Responses  int
	Mine       bool
	Responded  bool
}

// Audience selects who sees the status.
type Audience struct {
	// Users are concrete recipients; Community is an optional community id.
	Users     []int64
	Community *int64
}

// Create publishes a status. The author confirms the audience before sending;
// the server stores exactly that list.
func (s *Service) Create(ctx context.Context, authorID int64, text string, expiresAt time.Time, audience Audience) (Status, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return Status{}, httpapi.Validation(map[string]string{"text": "Добавьте текст статуса"}, "Проверьте поля")
	}
	if len([]rune(text)) > 280 {
		return Status{}, httpapi.Validation(map[string]string{"text": "Слишком длинный текст"}, "Проверьте поля")
	}
	// The status must end in the future: an expired one is useless on creation.
	now := time.Now()
	if !expiresAt.After(now.Add(time.Minute)) || expiresAt.After(now.Add(24*time.Hour)) {
		return Status{}, httpapi.Validation(map[string]string{"expires_at": "Выберите время от минуты до 24 часов в будущем"}, "Проверьте поля")
	}
	if len(audience.Users) > 100 || (audience.Community != nil && *audience.Community <= 0) {
		return Status{}, httpapi.Validation(map[string]string{"audience": "Выберите до 100 получателей"}, "Проверьте поля")
	}
	recipients := make([]int64, 0, len(audience.Users))
	seen := make(map[int64]bool)
	for _, id := range audience.Users {
		if id <= 0 {
			return Status{}, httpapi.Validation(map[string]string{"audience": "Некорректный получатель"}, "Проверьте поля")
		}
		if id != authorID && !seen[id] {
			recipients = append(recipients, id)
			seen[id] = true
		}
	}
	audience.Users = recipients
	if len(audience.Users) == 0 && audience.Community == nil {
		return Status{}, httpapi.Validation(map[string]string{"audience": "Выберите, кто увидит статус"}, "Проверьте поля")
	}

	var status Status
	err := s.pool.InTx(ctx, func(tx pgx.Tx) error {
		if err := validateAudience(ctx, tx, authorID, audience); err != nil {
			return err
		}
		if err := tx.QueryRow(ctx,
			`INSERT INTO pulse_statuses (author_id, text_sealed, key_id, expires_at)
			 VALUES ($1, ''::bytea, 'pending', $2)
			 RETURNING id, created_at`, authorID, expiresAt,
		).Scan(&status.ID, &status.CreatedAt); err != nil {
			return err
		}

		sealed, err := s.content.SealString(text, statusAAD(status.ID, authorID))
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx,
			`UPDATE pulse_statuses SET text_sealed = $2, key_id = $3 WHERE id = $1`,
			status.ID, sealed.Ciphertext, sealed.KeyID,
		); err != nil {
			return err
		}

		for _, userID := range audience.Users {
			if userID == authorID {
				continue
			}
			if _, err := tx.Exec(ctx,
				`INSERT INTO pulse_audience (status_id, user_id) VALUES ($1, $2)
				 ON CONFLICT DO NOTHING`, status.ID, userID,
			); err != nil {
				return err
			}
		}
		if audience.Community != nil {
			if _, err := tx.Exec(ctx,
				`INSERT INTO pulse_audience (status_id, community_id) VALUES ($1, $2)`,
				status.ID, *audience.Community,
			); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return Status{}, err
	}

	status.AuthorID = authorID
	status.Text = text
	status.ExpiresAt = expiresAt
	status.Mine = true
	return status, nil
}

// Feed returns the active statuses the viewer may see: their own, statuses
// addressed to them, and statuses of communities they belong to.
func (s *Service) Feed(ctx context.Context, viewerID int64) ([]Status, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT p.id, p.author_id, u.public_name, p.text_sealed, p.key_id,
		        p.expires_at, p.closed_at, p.created_at,
		        (SELECT count(*) FROM pulse_responses r WHERE r.status_id = p.id) AS responses,
		        p.author_id = $1 AS mine,
		        EXISTS (SELECT 1 FROM pulse_responses r2 WHERE r2.status_id = p.id AND r2.user_id = $1) AS responded
		   FROM pulse_statuses p
		   JOIN users u ON u.id = p.author_id
		  WHERE p.closed_at IS NULL
		    AND p.expires_at > now()
		    AND `+visibleStatusSQL+`
		  ORDER BY p.expires_at, p.id
		  LIMIT 100`, viewerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]Status, 0, 16)
	for rows.Next() {
		var status Status
		var sealed []byte
		var keyID string
		if err := rows.Scan(&status.ID, &status.AuthorID, &status.AuthorName, &sealed, &keyID,
			&status.ExpiresAt, &status.ClosedAt, &status.CreatedAt,
			&status.Responses, &status.Mine, &status.Responded,
		); err != nil {
			return nil, err
		}
		text, err := s.content.OpenString(secure.Sealed{KeyID: keyID, Ciphertext: sealed}, statusAAD(status.ID, status.AuthorID))
		if err != nil {
			return nil, err
		}
		status.Text = text
		out = append(out, status)
	}
	return out, rows.Err()
}

// Respond records the viewer's response once. A closed or expired status
// refuses, even if the client still shows it.
func (s *Service) Respond(ctx context.Context, viewerID, statusID int64) error {
	return s.pool.InTx(ctx, func(tx pgx.Tx) error {
		authorID, expiresAt, closedAt, err := lockVisibleStatus(ctx, tx, viewerID, statusID)
		if err != nil {
			return err
		}
		if authorID == viewerID {
			return httpapi.BadRequest("own_status", "Нельзя откликнуться на свой статус")
		}
		if closedAt != nil || !expiresAt.After(time.Now()) {
			return httpapi.Conflict("status_closed", "Статус уже завершён")
		}
		_, err = tx.Exec(ctx,
			`INSERT INTO pulse_responses (status_id, user_id) VALUES ($1, $2)
			 ON CONFLICT DO NOTHING`, statusID, viewerID)
		return err
	})
}

// Close ends the author's own status early.
func (s *Service) Close(ctx context.Context, authorID, statusID int64) error {
	tag, err := s.pool.Exec(ctx,
		`UPDATE pulse_statuses SET closed_at = now()
		  WHERE id = $1 AND author_id = $2 AND closed_at IS NULL`, statusID, authorID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return httpapi.NotFound("Статус не найден")
	}
	return nil
}

func statusAAD(statusID, authorID int64) string {
	return "pulse_status:" + itoa(statusID) + ":author:" + itoa(authorID)
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
