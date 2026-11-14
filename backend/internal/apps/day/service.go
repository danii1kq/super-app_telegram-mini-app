package day

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/danii1kq/srunal/internal/platform/httpapi"
	"github.com/danii1kq/srunal/internal/platform/postgres"
	"github.com/jackc/pgx/v5"
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
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return externalID != "", nil
}

// Week returns events for the caller's current week.
func (s *Service) Week(ctx context.Context, userID int64, initData string, from time.Time) ([]Event, error) {
	return s.Window(ctx, userID, initData, from, from.AddDate(0, 0, 7))
}

func (s *Service) Window(ctx context.Context, userID int64, initData string, from, to time.Time) ([]Event, error) {
	if !to.After(from) || to.Sub(from) > 8*24*time.Hour {
		return nil, httpapi.BadRequest("invalid_range", "Выберите период до восьми дней")
	}
	linked, err := s.Linked(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !linked {
		return nil, httpapi.Conflict("not_linked", "Аккаунт SruDay ещё не привязан")
	}
	return s.client.ListEvents(ctx, initData, from, to)
}

// CreateDeadline creates an event from a SruDoc order.
//
// This is the first cross-app action: SruNal verifies the caller owns the order
// through the doc adapter before this is called. The upstream does not support
// idempotency keys: an unknown POST result must be checked before a retry.
func (s *Service) CreateDeadline(ctx context.Context, userID int64, initData string, input CreateEventInput) (Event, error) {
	if input.AllDay && input.StartsAt == input.EndsAt {
		var zone string
		if err := s.pool.QueryRow(ctx, `SELECT timezone FROM user_settings WHERE user_id=$1`, userID).Scan(&zone); err != nil {
			return Event{}, err
		}
		var err error
		input, err = deadlineDay(input, zone)
		if err != nil {
			return Event{}, err
		}
	}
	return s.Create(ctx, userID, initData, input)
}

func deadlineDay(input CreateEventInput, zone string) (CreateEventInput, error) {
	start, err := time.Parse(time.RFC3339, input.StartsAt)
	if err != nil {
		return input, httpapi.BadRequest("invalid_date", "Некорректная дата дедлайна")
	}
	loc, err := time.LoadLocation(zone)
	if err != nil {
		return input, httpapi.BadRequest("invalid_timezone", "Некорректный часовой пояс")
	}
	local := start.In(loc)
	start = time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)
	input.StartsAt = start.Format(time.RFC3339)
	input.EndsAt = start.AddDate(0, 0, 1).Format(time.RFC3339)
	return input, nil
}

func validateEvent(input *CreateEventInput) error {
	input.Title = strings.TrimSpace(input.Title)
	input.Location = strings.TrimSpace(input.Location)
	start, startErr := time.Parse(time.RFC3339, input.StartsAt)
	end, endErr := time.Parse(time.RFC3339, input.EndsAt)
	if input.Title == "" || len([]rune(input.Title)) > 200 || len([]rune(input.Location)) > 200 || startErr != nil || endErr != nil || end.Sub(start) < time.Minute || end.Sub(start) > 30*24*time.Hour {
		return httpapi.Validation(map[string]string{"event": "Название до 200 символов, длительность от 1 минуты до 30 дней"}, "Проверьте событие")
	}
	return nil
}

// Create saves only a personal, private event after explicit confirmation.
func (s *Service) Create(ctx context.Context, userID int64, initData string, input CreateEventInput) (Event, error) {
	if err := validateEvent(&input); err != nil {
		return Event{}, err
	}
	linked, err := s.Linked(ctx, userID)
	if err != nil {
		return Event{}, err
	}
	if !linked {
		return Event{}, httpapi.Conflict("not_linked", "Аккаунт SruDay ещё не привязан")
	}
	return s.client.CreateEvent(ctx, initData, input)
}

// Connect stores the account returned by SruDay, never a client supplied ID.
func (s *Service) Connect(ctx context.Context, userID int64, initData string) error {
	session, err := s.client.authenticate(ctx, initData)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx,
		`INSERT INTO app_links (user_id, app_id, external_id) VALUES ($1, 'day', $2)
		 ON CONFLICT (user_id, app_id) DO UPDATE SET external_id = excluded.external_id`,
		userID, strconv.FormatInt(session.User.ID, 10),
	)
	return err
}

func (s *Service) Disconnect(ctx context.Context, userID int64, initData string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM app_links WHERE user_id=$1 AND app_id='day'`, userID)
	if err == nil {
		s.client.forget(initData)
	}
	return err
}
