package auth

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

// Settings is the per-user shell configuration.
type Settings struct {
	Lang     string
	Theme    string
	Timezone string
	Pinned   []string
}

// Settings reads the user's shell settings, falling back to defaults when the
// row is missing.
func (s *Service) Settings(ctx context.Context, userID int64) (Settings, error) {
	var out Settings
	err := s.pool.QueryRow(ctx,
		`SELECT lang, theme, timezone, pinned_apps FROM user_settings WHERE user_id = $1`, userID,
	).Scan(&out.Lang, &out.Theme, &out.Timezone, &out.Pinned)
	if errors.Is(err, pgx.ErrNoRows) {
		return Settings{Lang: "ru", Theme: "auto", Timezone: "Asia/Almaty"}, nil
	}
	if err != nil {
		return Settings{}, err
	}
	return out, nil
}
