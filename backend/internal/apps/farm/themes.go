package farm

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"

	"github.com/danii1kq/srunal/internal/platform/httpapi"
	"github.com/jackc/pgx/v5"
)

type Theme struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Stars       int    `json:"stars"`
}

var Themes = []Theme{
	{"garden", "Тихий сад", "Знакомая зелень и мягкое утреннее солнце.", 0},
	{"tropical", "Тропический остров", "Бирюзовая вода, пальмы и тёплый песок вокруг твоих грядок.", 150},
	{"lunar", "Лунная оранжерея", "Ночное небо, светящиеся растения и сад под луной.", 200},
}

func findTheme(id string) (Theme, bool) {
	for _, theme := range Themes {
		if theme.ID == id {
			return theme, true
		}
	}
	return Theme{}, false
}
func ownsTheme(state State, id string) bool {
	for _, owned := range state.OwnedThemes {
		if owned == id {
			return true
		}
	}
	return false
}
func unlockTheme(state *State, id string) {
	if !ownsTheme(*state, id) {
		state.OwnedThemes = append(state.OwnedThemes, id)
	}
	state.Theme = id
}
func codeHash(code string) string {
	sum := sha256.Sum256([]byte(strings.ToUpper(strings.TrimSpace(code))))
	return hex.EncodeToString(sum[:])
}

// Development codes are seeded only when insecure local development is enabled.
func (h *Handler) SeedDevThemes(ctx context.Context) error {
	for code, theme := range map[string]string{"TROPIC": "tropical", "LUNA": "lunar"} {
		if _, err := h.pool.Exec(ctx, `INSERT INTO farm_theme_codes(code_hash,theme,max_uses) VALUES ($1,$2,100000) ON CONFLICT DO NOTHING`, codeHash(code), theme); err != nil {
			return err
		}
	}
	return nil
}

func redeemTheme(ctx context.Context, tx pgx.Tx, userID int64, state *State, code string) error {
	if len(code) > 64 || strings.TrimSpace(code) == "" {
		return httpapi.BadRequest("invalid_code", "Введи промокод")
	}
	hash := codeHash(code)
	var theme string
	var available bool
	err := tx.QueryRow(ctx, `SELECT theme,uses<max_uses AND (expires_at IS NULL OR expires_at>now()) FROM farm_theme_codes WHERE code_hash=$1 FOR UPDATE`, hash).Scan(&theme, &available)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && !available) {
		return httpapi.BadRequest("invalid_code", "Промокод не найден или уже закончился")
	}
	if err != nil {
		return err
	}
	var used bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM farm_theme_redemptions WHERE code_hash=$1 AND user_id=$2)`, hash, userID).Scan(&used); err != nil {
		return err
	}
	if used {
		return httpapi.BadRequest("used_code", "Ты уже активировал этот промокод")
	}
	if ownsTheme(*state, theme) {
		return httpapi.BadRequest("theme_owned", "Эта тема у тебя уже есть")
	}
	if _, err = tx.Exec(ctx, `INSERT INTO farm_theme_redemptions(code_hash,user_id) VALUES ($1,$2)`, hash, userID); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `UPDATE farm_theme_codes SET uses=uses+1 WHERE code_hash=$1`, hash); err != nil {
		return err
	}
	unlockTheme(state, theme)
	return nil
}
