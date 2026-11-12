package day

import (
	"context"
	"time"

	"github.com/danii1kq/srunal/internal/platform/httpapi"
	"github.com/danii1kq/srunal/internal/platform/postgres"
)

// Service bridges the SruNal account to SruDay.
type Service struct {
	pool   *postgres.Pool
	client *Client
}

func NewService(pool *postgres.Pool, client *Client) *Service {
	return &Service{pool: pool, client: client}
}

// Linked reports whether the account is connected to SruDay.
func (s *Service) Linked(ctx context.Context, userID int64) (bool, error) {
	var externalID string
	err := s.pool.QueryRow(ctx,
		`SELECT external_id FROM app_links WHERE user_id = $1 AND app_id = 'day'`, userID,
	).Scan(&externalID)
	if err != nil {
		return false, nil
	}
	return externalID != "", nil
}

// Week returns events for the caller's current week.
func (s *Service) Week(ctx context.Context, userID int64, initData string, from time.Time) ([]Event, error) {
	linked, err := s.Linked(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !linked {
		return nil, httpapi.Conflict("not_linked", "Аккаунт SruDay ещё не привязан")
	}
	return s.client.ListEvents(ctx, initData, from, from.AddDate(0, 0, 7))
}

// CreateDeadline creates an event from a SruDoc order.
//
// This is the first cross-app action: SruNal verifies the caller owns the order
// through the doc adapter before this is called, and source_ref makes the
// creation idempotent on SruDay's side.
func (s *Service) CreateDeadline(ctx context.Context, userID int64, initData string, input CreateEventInput) (Event, error) {
	linked, err := s.Linked(ctx, userID)
	if err != nil {
		return Event{}, err
	}
	if !linked {
		return Event{}, httpapi.Conflict("not_linked", "Аккаунт SruDay ещё не привязан")
	}
	return s.client.CreateEvent(ctx, initData, input)
}

// Link stores a confirmed SruDay account id.
func (s *Service) Link(ctx context.Context, userID int64, externalID string) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO app_links (user_id, app_id, external_id) VALUES ($1, 'day', $2)
		 ON CONFLICT (user_id, app_id) DO UPDATE SET external_id = excluded.external_id`,
		userID, externalID,
	)
	return err
}
