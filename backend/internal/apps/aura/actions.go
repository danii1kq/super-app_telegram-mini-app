package aura

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/danii1kq/srunal/internal/platform/httpapi"
)

// Result is what one action returns. The client renders it as a line of
// feedback plus the refreshed hero.
type Result struct {
	Message  string `json:"message"`
	Correct  bool   `json:"correct"`
	Answer   int    `json:"answer,omitempty"`
	Hint     string `json:"hint,omitempty"`
	AuraGain int64  `json:"aura_gain"`
	XPGain   int    `json:"xp_gain"`
	Damage   int    `json:"damage"`
	LevelUp  int    `json:"level_up"`
	Relic    *Relic `json:"relic,omitempty"`
	Finished bool   `json:"finished"`
	Player   Player `json:"player"`
}

// farmEvents are the flavour lines. They are the reason the loop feels like a
// student's day rather than a spreadsheet.
var farmEvents = []string{
	"Ты сдал лабу с первого раза",
	"Ты нашёл свободную аудиторию для сна",
	"Ты успел на автобус, который уже отъезжал",
	"Ты принёс воду на пару и стал героем",
	"Ты заказал шаурму, а привезли две",
	"Ты угадал, какой препод зайдёт",
	"Ты нашёл в кармане 500 тенге",
	"Ты сдал курсовую за ночь",
	"Ты объяснил преподу, что интернет не работал",
	"Ты выиграл катку на последней секунде",
}

// Farm collects the passive aura. It is the slow loop: four hours between
// collections, and the streak grows while the player keeps coming back.
func (s *Service) Farm(ctx context.Context, userID int64) (Result, error) {
	var result Result
	err := s.pool.InTx(ctx, func(tx pgx.Tx) error {
		data, err := s.locked(ctx, tx, userID)
		if err != nil {
			return err
		}
		now := s.now()
		if err := requireReady(data.FarmReadyAt, now, "Аура"); err != nil {
			return err
		}

		relics, err := s.wornRelicsTx(ctx, tx, userID)
		if err != nil {
			return err
		}
		base := int64(300)
		base += int64(float64(base) * relicBonus(relics, "farm"))
		base += int64(float64(base) * bonusFor(data.HeroClass).Farm)

		// The streak rewards showing up on consecutive days.
		today := now.Truncate(24 * time.Hour)
		if data.LastSeenDay == nil || !sameDay(*data.LastSeenDay, now) {
			if data.LastSeenDay != nil && sameDay(data.LastSeenDay.AddDate(0, 0, 1), now) {
				data.Streak++
			} else {
				data.Streak = 1
			}
			data.LastSeenDay = &today
		}
		streakBonus := int64(data.Streak) * 10
		if streakBonus > 200 {
			streakBonus = 200
		}
		gain := base + streakBonus

		data.Aura += gain
		data.TotalAura += gain
		ready := now.Add(FarmCooldown)
		data.FarmReadyAt = &ready

		level, gained := grantXP(&data, 15)
		result = Result{
			Message:  farmEvents[s.rnd.Intn(len(farmEvents))],
			AuraGain: gain,
			XPGain:   15,
			LevelUp:  gained,
		}
		_ = level

		if err := s.save(ctx, tx, userID, data); err != nil {
			return err
		}
		return s.record(ctx, tx, userID, "farm", fmt.Sprintf("farm-%d", now.UnixNano()), gain, 15, 0, 0)
	})
	if err != nil {
		return Result{}, err
	}
	player, err := s.snapshot(ctx, userID)
	if err != nil {
		return Result{}, err
	}
	result.Player = player
	return result, nil
}

// Start opens a challenge. Only one may be open at a time, which is why the
// session lives in the row: a second start would otherwise overwrite a live run.
func (s *Service) Start(ctx context.Context, userID int64, kind SessionKind) (Result, error) {
	var result Result
	err := s.pool.InTx(ctx, func(tx pgx.Tx) error {
		data, err := s.locked(ctx, tx, userID)
		if err != nil {
			return err
		}
		now := s.now()

		if data.Session != nil && now.Before(data.Session.ExpiresAt) {
			return bad("Сначала закончи текущее испытание")
		}
		data.Session = nil

		relics, err := s.wornRelicsTx(ctx, tx, userID)
		if err != nil {
			return err
		}
		_ = relics

		session := &Session{
			Kind:      kind,
			AskedAt:   now,
			ExpiresAt: now.Add(SessionTTL),
			Pity:      data.GrindToday,
		}

		switch kind {
		case SessionArena:
			if err := requireReady(data.ArenaReadyAt, now, "Арена"); err != nil {
				return err
			}
			boss, hp := arenaBoss(s.rnd, data.Level)
			session.Rounds = ArenaRounds
			session.Round = 1
			session.BossName = boss
			session.BossHP = hp
			session.BossMaxHP = hp
			session.Seconds = secondsForQuestion(data.Level, data.HeroClass)
		case SessionGate:
			if err := requireReady(data.GateReadyAt, now, "Подземелье"); err != nil {
				return err
			}
			boss, hp := arenaBoss(s.rnd, data.Level+5)
			session.BossName = boss
			session.BossHP = hp
			session.BossMaxHP = hp
			session.Rounds = 5
			session.Round = 1
			session.Seconds = secondsForQuestion(data.Level, data.HeroClass)
		case SessionTower:
			if err := requireReady(data.TowerReadyAt, now, "Башня"); err != nil {
				return err
			}
			floor := data.Level/5 + 1
			session.Floor = floor
			session.BossName = fmt.Sprintf("Страж этажа %d", floor)
			session.BossHP = 50 + floor*30
			session.BossMaxHP = session.BossHP
			session.Rounds = 3
			session.Round = 1
			session.Seconds = secondsForQuestion(data.Level, data.HeroClass)
		case SessionGrind:
			if data.GrindToday >= GrindDailyCap {
				return bad("На сегодня вылазок хватит, возвращайся завтра")
			}
			session.Rounds = 1
			session.Round = 1
			session.Seconds = 15
		default:
			return bad("Неизвестное испытание")
		}

		session.Question = Generate(s.rnd, data.Level, "")
		data.Session = session
		result = Result{Message: describeSession(session)}
		if err := s.save(ctx, tx, userID, data); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return Result{}, err
	}
	player, err := s.snapshot(ctx, userID)
	if err != nil {
		return Result{}, err
	}
	result.Player = player
	return result, nil
}

// Answer resolves the open question. The answer is checked on the server: the
// client never sends whether it was right.
func (s *Service) Answer(ctx context.Context, userID int64, choice int, requestID string) (Result, error) {
	if requestID == "" {
		return Result{}, bad("Не хватает номера запроса")
	}
	var result Result
	err := s.pool.InTx(ctx, func(tx pgx.Tx) error {
		data, err := s.locked(ctx, tx, userID)
		if err != nil {
			return err
		}
		now := s.now()

		session := data.Session
		if session == nil {
			return errNoSession
		}
		if now.After(session.ExpiresAt) {
			data.Session = nil
			if err := s.save(ctx, tx, userID, data); err != nil {
				return err
			}
			return bad("Время вышло, начни заново")
		}

		relics, err := s.wornRelicsTx(ctx, tx, userID)
		if err != nil {
			return err
		}

		correct := choice == session.Question.Answer
		result.Correct = correct
		result.Answer = session.Question.Answer

		if correct {
			session.Correct++
			session.Combo++
			if session.Combo > session.BestCombo {
				session.BestCombo = session.Combo
			}
			damage := 10 + data.Stats.Str*2 + session.Combo*2
			damage += int(float64(damage) * relicBonus(relics, "damage"))
			damage += int(float64(damage) * bonusFor(data.HeroClass).ArenaDamage)
			if s.rnd.Float64() < critChance(data.Stats, relics) {
				damage *= 2
			}
			session.Damage += damage
			session.BossHP -= damage
			result.Damage = damage
		} else {
			session.Combo = 0
			result.Hint = hint(session.Question.Topic)
		}

		session.Round++
		finished := session.Round > session.Rounds || session.BossHP <= 0

		if !finished {
			session.Question = Generate(s.rnd, data.Level, "")
			session.AskedAt = now
			session.ExpiresAt = now.Add(SessionTTL)
			data.Session = session
			result.Message = describeSession(session)
			if err := s.save(ctx, tx, userID, data); err != nil {
				return err
			}
			return s.record(ctx, tx, userID, kindOf(session.Kind), requestID, 0, 0, result.Damage, boolInt(correct))
		}

		// The run is over: pay out, clear the session and set the cooldown.
		data.Session = nil
		gain, xp, relic := s.reward(&data, session, now)
		data.Aura += gain
		data.TotalAura += gain
		level, gained := grantXP(&data, xp)
		_ = level
		result.AuraGain = gain
		result.XPGain = xp
		result.LevelUp = gained
		result.Finished = true
		result.Relic = relic
		result.Message = fmt.Sprintf("%s завершено: +%d ауры", kindLabel(session.Kind), gain)

		if relic != nil {
			if err := s.grantRelic(ctx, tx, userID, *relic); err != nil {
				return err
			}
		}
		if err := s.save(ctx, tx, userID, data); err != nil {
			return err
		}
		return s.record(ctx, tx, userID, kindOf(session.Kind), requestID, gain, xp, result.Damage, boolInt(correct))
	})
	if err != nil {
		if err == errNoSession {
			return Result{}, bad("Нет открытого испытания")
		}
		return Result{}, err
	}
	player, err := s.snapshot(ctx, userID)
	if err != nil {
		return Result{}, err
	}
	result.Player = player
	return result, nil
}

// reward computes the payout for a finished run.
func (s *Service) reward(data *row, session *Session, now time.Time) (int64, int, *Relic) {
	switch session.Kind {
	case SessionArena:
		ready := now.Add(ArenaCooldown)
		data.ArenaReadyAt = &ready
		gain := int64(40 + session.Correct*25 + session.BestCombo*10)
		xp := 20 + session.Correct*5
		var relic *Relic
		// A clean run is every question answered. The boss can die before the
		// last round, so comparing against Rounds would almost never hold.
		if session.Correct >= session.Round-1 {
			picked := rollRelic(s.rnd, rollRank(s.rnd, RankD))
			relic = &picked
		}
		return gain, xp, relic
	case SessionGate:
		ready := now.Add(GateCooldown)
		data.GateReadyAt = &ready
		gain := int64(80 + session.Correct*30)
		xp := 30 + session.Correct*6
		picked := rollRelic(s.rnd, rollRank(s.rnd, RankC))
		return gain, xp, &picked
	case SessionTower:
		ready := now.Add(TowerCooldown)
		data.TowerReadyAt = &ready
		gain := int64(30 + session.Floor*15)
		xp := 15 + session.Floor*3
		var relic *Relic
		// Every fifth floor is a guaranteed relic, which gives the tower a
		// reason to be climbed rather than ground.
		if session.Floor%5 == 0 {
			picked := rollRelic(s.rnd, rollRank(s.rnd, RankB))
			relic = &picked
		}
		return gain, xp, relic
	case SessionGrind:
		multiplier := grindMultiplier(data.GrindToday)
		data.GrindToday++
		gain := int64(float64(15+session.Correct*10) * multiplier)
		xp := int(float64(5+session.Correct*2) * multiplier)
		return gain, xp, nil
	}
	return 0, 0, nil
}

// grantRelic stores a relic, awakening it instead when the player already owns
// it. A duplicate is never wasted: it raises the level, and past level five it
// pays aura.
func (s *Service) grantRelic(ctx context.Context, tx pgx.Tx, userID int64, relic Relic) error {
	var level int
	err := tx.QueryRow(ctx,
		`SELECT level FROM aura_relics WHERE user_id = $1 AND relic_key = $2 FOR UPDATE`,
		userID, relic.Key).Scan(&level)
	if err == pgx.ErrNoRows {
		_, err = tx.Exec(ctx,
			`INSERT INTO aura_relics (user_id, relic_key, rank) VALUES ($1, $2, $3)`,
			userID, relic.Key, relic.Rank)
		return err
	}
	if err != nil {
		return err
	}
	if level < 5 {
		_, err = tx.Exec(ctx,
			`UPDATE aura_relics SET level = level + 1 WHERE user_id = $1 AND relic_key = $2`,
			userID, relic.Key)
		return err
	}
	// Maxed: pay compensation so the pull is not a dead result.
	_, err = tx.Exec(ctx,
		`UPDATE aura_players SET aura = aura + 200, total_aura = total_aura + 200 WHERE user_id = $1`,
		userID)
	return err
}

// record writes the run row. The unique request id makes a retry a no-op.
func (s *Service) record(ctx context.Context, tx pgx.Tx, userID int64, kind, requestID string, auraGain int64, xpGain, damage, correct int) error {
	_, err := tx.Exec(ctx,
		`INSERT INTO aura_runs (user_id, kind, request_id, aura_gain, xp_gain, damage, correct)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 ON CONFLICT (user_id, request_id) DO NOTHING`,
		userID, kind, requestID, auraGain, xpGain, damage, correct)
	return err
}

// locked reads the hero inside a transaction with the row locked, so two
// simultaneous answers cannot both spend the same session.
func (s *Service) locked(ctx context.Context, tx pgx.Tx, userID int64) (row, error) {
	if _, err := tx.Exec(ctx,
		`INSERT INTO aura_players (user_id) VALUES ($1) ON CONFLICT DO NOTHING`, userID); err != nil {
		return row{}, err
	}
	var out row
	var rawSession []byte
	var grindDay, lastSeen *time.Time
	err := tx.QueryRow(ctx,
		`SELECT aura, total_aura, level, xp, stat_points,
		        stat_str, stat_agi, stat_int, stat_vit, stat_per,
		        hero_class, farm_ready_at, arena_ready_at, gate_ready_at, tower_ready_at,
		        grind_day, grind_today, session, streak, last_seen_day
		   FROM aura_players WHERE user_id = $1 FOR UPDATE`, userID,
	).Scan(&out.Aura, &out.TotalAura, &out.Level, &out.XP, &out.StatPoints,
		&out.Stats.Str, &out.Stats.Agi, &out.Stats.Int, &out.Stats.Vit, &out.Stats.Per,
		&out.HeroClass, &out.FarmReadyAt, &out.ArenaReadyAt, &out.GateReadyAt, &out.TowerReadyAt,
		&grindDay, &out.GrindToday, &rawSession, &out.Streak, &lastSeen)
	if err != nil {
		return row{}, err
	}
	out.GrindDay = grindDay
	out.LastSeenDay = lastSeen
	session, err := sessionFromJSON(rawSession)
	if err != nil {
		return row{}, err
	}
	out.Session = session
	return out, nil
}

// wornRelicsTx is wornRelics inside a transaction.
func (s *Service) wornRelicsTx(ctx context.Context, tx pgx.Tx, userID int64) ([]Relic, error) {
	rows, err := tx.Query(ctx,
		`SELECT relic_key FROM aura_relics WHERE user_id = $1 AND slot <> ''`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Relic, 0, 3)
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, err
		}
		if relic, ok := relicByKey(key); ok {
			out = append(out, relic)
		}
	}
	return out, rows.Err()
}

func kindOf(kind SessionKind) string {
	return string(kind)
}

func kindLabel(kind SessionKind) string {
	switch kind {
	case SessionArena:
		return "Арена"
	case SessionGate:
		return "Подземелье"
	case SessionTower:
		return "Башня"
	case SessionGrind:
		return "Вылазка"
	}
	return "Испытание"
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func sameDay(a, b time.Time) bool {
	ay, am, ad := a.UTC().Date()
	by, bm, bd := b.UTC().Date()
	return ay == by && am == bm && ad == bd
}

var _ = httpapi.BadRequest
