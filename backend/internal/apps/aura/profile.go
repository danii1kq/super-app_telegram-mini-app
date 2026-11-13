package aura

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/danii1kq/srunal/internal/platform/httpapi"
)

// OwnedRelic is a relic the player has, with its worn slot and level.
type OwnedRelic struct {
	Relic
	Level int  `json:"level"`
	Worn  bool `json:"worn"`
}

// relics lists everything the player owns, worn first.
func (s *Service) relics(ctx context.Context, userID int64) ([]OwnedRelic, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT relic_key, level, slot FROM aura_relics WHERE user_id = $1
		  ORDER BY (slot <> '') DESC, level DESC, obtained_at`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]OwnedRelic, 0, 8)
	for rows.Next() {
		var key, slot string
		var level int
		if err := rows.Scan(&key, &level, &slot); err != nil {
			return nil, err
		}
		relic, ok := relicByKey(key)
		if !ok {
			continue
		}
		out = append(out, OwnedRelic{Relic: relic, Level: level, Worn: slot != ""})
	}
	return out, rows.Err()
}

// ChooseClass sets the hero path. The first choice is free; a reroll costs
// aura, which is what stops a player from flipping paths every run.
func (s *Service) ChooseClass(ctx context.Context, userID int64, class HeroClass) (Player, error) {
	if _, ok := classBonuses[class]; !ok {
		return Player{}, httpapi.BadRequest("aura", "Такого пути нет")
	}
	err := s.pool.InTx(ctx, func(tx pgx.Tx) error {
		data, err := s.locked(ctx, tx, userID)
		if err != nil {
			return err
		}
		if data.HeroClass != "" {
			if data.Aura < ClassRerollCost {
				return httpapi.BadRequest("aura", "Для смены пути нужно 300 ауры")
			}
			data.Aura -= ClassRerollCost
		}
		data.HeroClass = class
		return s.save(ctx, tx, userID, data)
	})
	if err != nil {
		return Player{}, err
	}
	return s.snapshot(ctx, userID)
}

// Equip wears a relic, or takes it off when slot is empty.
//
// Wearing one relic into a slot replaces whatever was there, because the unique
// index allows only one per slot and a player expects a swap rather than an
// error.
func (s *Service) Equip(ctx context.Context, userID int64, key string, slot Slot) error {
	relic, ok := relicByKey(key)
	if !ok {
		return httpapi.BadRequest("aura", "Такой реликвии нет")
	}
	if slot != "" && slot != relic.Slot {
		return httpapi.BadRequest("aura", "Эта реликвия не подходит для этого слота")
	}
	return s.pool.InTx(ctx, func(tx pgx.Tx) error {
		var owned bool
		if err := tx.QueryRow(ctx,
			`SELECT EXISTS (SELECT 1 FROM aura_relics WHERE user_id = $1 AND relic_key = $2)`,
			userID, key).Scan(&owned); err != nil {
			return err
		}
		if !owned {
			return httpapi.BadRequest("aura", "Этой реликвии у тебя нет")
		}
		if slot == "" {
			_, err := tx.Exec(ctx,
				`UPDATE aura_relics SET slot = '' WHERE user_id = $1 AND relic_key = $2`,
				userID, key)
			return err
		}
		// Clear the slot first: the unique index forbids two rows with the same
		// slot, so the swap has to happen in two steps.
		if _, err := tx.Exec(ctx,
			`UPDATE aura_relics SET slot = '' WHERE user_id = $1 AND slot = $2`,
			userID, slot); err != nil {
			return err
		}
		_, err := tx.Exec(ctx,
			`UPDATE aura_relics SET slot = $3 WHERE user_id = $1 AND relic_key = $2`,
			userID, key, slot)
		return err
	})
}

// SpendStat puts one point into an attribute. Points come from levels, so a
// player cannot buy them.
func (s *Service) SpendStat(ctx context.Context, userID int64, stat string) (Player, error) {
	err := s.pool.InTx(ctx, func(tx pgx.Tx) error {
		data, err := s.locked(ctx, tx, userID)
		if err != nil {
			return err
		}
		if data.StatPoints <= 0 {
			return httpapi.BadRequest("aura", "Нет свободных очков")
		}
		switch stat {
		case "str":
			data.Stats.Str++
		case "agi":
			data.Stats.Agi++
		case "int":
			data.Stats.Int++
		case "vit":
			data.Stats.Vit++
		case "per":
			data.Stats.Per++
		default:
			return httpapi.BadRequest("aura", "Такого параметра нет")
		}
		data.StatPoints--
		return s.save(ctx, tx, userID, data)
	})
	if err != nil {
		return Player{}, err
	}
	return s.snapshot(ctx, userID)
}

// TopEntry is one row of the leaderboard.
type TopEntry struct {
	UserID    string `json:"user_id"`
	Name      string `json:"name"`
	Level     int    `json:"level"`
	TotalAura int64  `json:"total_aura"`
	Rank      int    `json:"rank"`
	Mine      bool   `json:"mine"`
}

// Top ranks by lifetime aura, which is the number that only grows.
func (s *Service) Top(ctx context.Context, viewerID int64, limit int) ([]TopEntry, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	rows, err := s.pool.Query(ctx,
		`WITH ranked AS (
		     SELECT p.user_id, u.public_name, p.level, p.total_aura,
		            row_number() OVER (ORDER BY p.total_aura DESC, p.user_id) AS rank
		       FROM aura_players p JOIN users u ON u.id = p.user_id
		      WHERE u.status = 'active' AND p.total_aura > 0
		 )
		 SELECT user_id, public_name, level, total_aura, rank
		   FROM ranked WHERE rank <= $1 OR user_id = $2
		  ORDER BY rank`, limit, viewerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]TopEntry, 0, limit)
	for rows.Next() {
		var id int64
		var entry TopEntry
		if err := rows.Scan(&id, &entry.Name, &entry.Level, &entry.TotalAura, &entry.Rank); err != nil {
			return nil, err
		}
		entry.UserID = itoa(id)
		entry.Mine = id == viewerID
		out = append(out, entry)
	}
	return out, rows.Err()
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
