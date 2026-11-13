package aura

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/danii1kq/srunal/internal/platform/httpapi"
)

// Result is what one action returns. The client renders it as a line of
// feedback plus the refreshed hero.
type Result struct {
	Message       string         `json:"message"`
	Correct       bool           `json:"correct"`
	Answer        int            `json:"answer,omitempty"`
	Hint          string         `json:"hint,omitempty"`
	AuraGain      int64          `json:"aura_gain"`
	XPGain        int            `json:"xp_gain"`
	Damage        int            `json:"damage"`
	LevelUp       int            `json:"level_up"`
	Relic         *Relic         `json:"relic,omitempty"`
	Finished      bool           `json:"finished"`
	Player        Player         `json:"player"`
	Won           bool           `json:"won"`
	HeroDamage    int            `json:"hero_damage"`
	MaterialsGain map[string]int `json:"materials_gain,omitempty"`
	BonusRelics   []Relic        `json:"bonus_relics,omitempty"`
}

// Meditation messages share the same System voice as the dungeon interface.
var farmEvents = []string{
	"Система: ядро ауры стабилизировано",
	"Руны откликнулись на твою медитацию",
	"Система: накопленная энергия извлечена",
	"За закрытыми вратами стало тише. Аура восстановлена",
	"След пробуждения усилился",
	"Система: охотник готов к следующему контракту",
	"Отголоски разлома превратились в ауру",
	"Дыхание выровнялось. Ядро вновь наполнено",
	"Система: печать накопления снята",
	"Тусклые руны снова засияли",
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

// Start preserves the original service entry point.
func (s *Service) Start(ctx context.Context, userID int64, kind SessionKind) (Result, error) {
	return s.StartDungeon(ctx, userID, kind, "", "")
}
func (s *Service) StartDungeon(ctx context.Context, userID int64, kind SessionKind, dungeon, difficulty string) (Result, error) {
	result := Result{}
	err := s.pool.InTx(ctx, func(tx pgx.Tx) error {
		data, err := s.locked(ctx, tx, userID)
		if err != nil {
			return err
		}
		now := s.now()
		if data.Session != nil && now.Before(data.Session.ExpiresAt) {
			return bad("Сначала заверши текущее испытание")
		}
		if difficulty != "" && !validDifficulty(difficulty) {
			return bad("Выбери сложность примеров")
		}
		if difficulty == "" {
			difficulty = data.Progress.Difficulty
		}
		if kind == SessionGate {
			if dungeon == "" {
				dungeon = "crypt"
			}
			d, ok := dungeonByID(dungeon)
			if !ok {
				return bad("Такого подземелья нет")
			}
			if data.Level < d.Level {
				return bad(fmt.Sprintf("Нужен %d уровень охотника", d.Level))
			}
		}
		switch kind {
		case SessionGate:
			err = requireReady(data.GateReadyAt, now, "Врата")
		case SessionArena:
			err = requireReady(data.ArenaReadyAt, now, "Арена")
		case SessionTower:
			err = requireReady(data.TowerReadyAt, now, "Башня")
		case SessionGrind:
			if data.GrindToday >= GrindDailyCap {
				return bad("Награды вылазок обновятся завтра")
			}
		default:
			return bad("Неизвестное испытание")
		}
		if err != nil {
			return err
		}
		relics, err := s.wornRelicsTx(ctx, tx, userID)
		if err != nil {
			return err
		}
		id, err := newID()
		if err != nil {
			return err
		}
		session := &Session{ID: id, Kind: kind, Room: 1, Rooms: 1, Phase: "combat", Dungeon: dungeon, Difficulty: difficulty, Round: 1, Rounds: CombatRounds, HeroStats: data.Stats, HeroPath: data.HeroClass, Worn: relics, HeroMaxHP: heroMaxHP(data.Stats), HeroHP: heroMaxHP(data.Stats)}
		session.Sets = activeSetIDs(relics)
		if kind == SessionGate {
			session.Rooms = 3
			planDungeonEvent(session, s.rnd)
		}
		if kind == SessionTower {
			session.Floor = data.Progress.TowerFloor + 1
		}
		if err := s.enterRoom(session, "normal", now); err != nil {
			return err
		}
		if kind == SessionGrind {
			session.Rounds = 1
			session.BossHP = 1
			session.BossMaxHP = 1
			session.BossName = "Рунный осколок"
		}
		data.Session = session
		result.Message = "Система: испытание началось"
		return s.save(ctx, tx, userID, data)
	})
	return s.resultPlayer(ctx, userID, result, err)
}

func (s *Service) Answer(ctx context.Context, userID int64, choice int, requestID string, questionIDs ...string) (Result, error) {
	result := Result{}
	err := s.pool.InTx(ctx, func(tx pgx.Tx) error {
		data, err := s.locked(ctx, tx, userID)
		if err != nil {
			return err
		}
		if err = s.checkRequest(ctx, tx, userID, requestID); err != nil {
			return err
		}
		session := data.Session
		if session == nil {
			return bad("Нет открытого испытания")
		}
		if session.Phase != "combat" {
			return bad("Сначала выбери следующий путь")
		}
		if len(questionIDs) == 0 || questionIDs[0] != session.QuestionID {
			return bad("Этот пример уже сменился. Обнови состояние")
		}
		now := s.now()
		if !now.Before(session.ExpiresAt) {
			result = s.finishRun(&data, session, false, now)
		} else {
			timely := !now.After(session.QuestionDeadline)
			valid := false
			for _, option := range session.Question.Options {
				if option == choice {
					valid = true
				}
			}
			if timely && !valid {
				return bad("Выбери один из вариантов ответа")
			}
			result.Correct = timely && choice == session.Question.Answer
			result.Answer = session.Question.Answer
			if result.Correct {
				session.Correct++
				session.Combo++
				data.Progress.Correct++
				session.BestCombo = max(session.BestCombo, session.Combo)
				session.Charge = min(3, session.Charge+1)
				session.AbilityReady = session.Charge >= 3
				empowered := session.Empowered
				damage := calculateHit(session, s.rnd.Float64())
				session.Damage += damage
				session.BossHP = max(0, session.BossHP-damage)
				result.Damage = damage
				if session.Room == session.Rooms && session.BossHP <= 0 {
					session.EmpoweredFinisher = empowered
				}
				if acceptedEvent(session) && session.Event.ID == "red_gate" {
					result.HeroDamage = 2
					session.HeroHP = max(0, session.HeroHP-2)
				}
			} else {
				session.Errors++
				session.Combo = 0
				eventMistake(session)
				curseMistake(session)
				if !acceptedEvent(session) && session.Dungeon == "void" && session.Room == session.Rooms {
					session.BossHP = min(session.BossMaxHP, session.BossHP+8)
					session.Charge = max(0, session.Charge-1)
					session.AbilityReady = session.Charge >= 3
				}
				result.HeroDamage = incomingHit(session, s.rnd.Float64())
				session.HeroHP = max(0, session.HeroHP-result.HeroDamage)
				result.Hint = hint(session.Question.Topic)
				if !timely {
					result.Hint = "Время на пример вышло. Противник успел атаковать"
				}
			}
			session.Round++
			switch {
			case session.HeroHP <= 0 || (session.Round > session.Rounds && session.BossHP > 0):
				completed := s.finishRun(&data, session, false, now)
				completed.Correct = result.Correct
				completed.Answer = result.Answer
				completed.Damage = result.Damage
				completed.HeroDamage = result.HeroDamage
				completed.Hint = result.Hint
				result = completed
			case session.BossHP <= 0:
				if session.Kind == SessionGate {
					d, _ := dungeonByID(session.Dungeon)
					amount := 2 + session.Room
					if session.Elite {
						amount += 2
					}
					amount += session.HeroStats.Per / 5
					data.Progress.Materials[d.Material] += amount
					result.MaterialsGain = map[string]int{d.Material: amount}
				}
				if session.Room < session.Rooms {
					session.Phase = "choice"
					session.AbilityReady = session.Charge >= 3
					heal := int(float64(session.HeroMaxHP) * relicBonus(session.Worn, "heal"))
					applyHealing(session, heal)
					session.ExpiresAt = now.Add(SessionTTL)
					result.Message = "Комната зачищена. Материалы сохранены. Выбери путь"
					if revealDungeonEvent(session, now) {
						result.Message = "Система: обнаружена аномалия. " + session.Event.Name
					}
				} else {
					completed := s.finishRun(&data, session, true, now)
					completed.Correct = result.Correct
					completed.Answer = result.Answer
					completed.Damage = result.Damage
					if completed.MaterialsGain == nil {
						completed.MaterialsGain = map[string]int{}
					}
					for material, amount := range result.MaterialsGain {
						completed.MaterialsGain[material] += amount
					}
					result = completed
				}
			default:
				if err = s.nextQuestion(session, now); err != nil {
					return err
				}
				result.Message = "Верный удар"
				if !result.Correct {
					result.Message = "Противник атакует"
				}
			}
		}
		if result.Won && session.Kind == SessionGate {
			if err = s.recordWeekly(ctx, tx, userID, session, now); err != nil {
				return err
			}
		}
		if result.Relic != nil {
			compensation, err := s.grantRelic(ctx, tx, userID, *result.Relic)
			if err != nil {
				return err
			}
			data.Aura += compensation
			data.TotalAura += compensation
			result.AuraGain += compensation
		}
		for _, item := range result.BonusRelics {
			compensation, err := s.grantRelic(ctx, tx, userID, item)
			if err != nil {
				return err
			}
			data.Aura += compensation
			data.TotalAura += compensation
			result.AuraGain += compensation
		}
		if err = s.save(ctx, tx, userID, data); err != nil {
			return err
		}
		return s.record(ctx, tx, userID, string(session.Kind), requestID, result.AuraGain, result.XPGain, result.Damage, boolInt(result.Correct))
	})
	return s.resultPlayer(ctx, userID, result, err)
}
func (s *Service) finishRun(data *row, session *Session, won bool, now time.Time) Result {
	won = won && session.BossHP <= 0 && (session.Kind != SessionGate || session.Room >= session.Rooms)
	result := Result{Finished: true, Won: won, Message: "Поход завершён поражением. Собранные материалы сохранены"}
	data.Session = nil
	wait := 2 * time.Minute
	if won {
		switch session.Kind {
		case SessionArena:
			wait = ArenaCooldown
		case SessionGate:
			wait = GateCooldown
		default:
			wait = TowerCooldown
		}
	}
	ready := now.Add(wait)
	switch session.Kind {
	case SessionGate:
		data.GateReadyAt = &ready
	case SessionArena:
		data.ArenaReadyAt = &ready
	case SessionTower:
		data.TowerReadyAt = &ready
	case SessionGrind:
		data.GrindToday++
	}
	if !won {
		return result
	}
	gain, xp, relic := s.reward(data, session, now)
	if session.DoubleReward {
		gain *= 2
		xp *= 2
	}
	xp += int(float64(xp) * relicBonus(session.Worn, "xp"))
	data.Aura += gain
	data.TotalAura += gain
	_, levels := grantXP(data, xp)
	result.AuraGain = gain
	result.XPGain = xp
	result.LevelUp = levels
	if relic != nil {
		view := scaledRelic(*relic, 1)
		result.Relic = &view
	}
	result.Message = fmt.Sprintf("Победа! +%d ауры, +%d опыта", gain, xp)
	if session.Kind == SessionGate {
		result.BonusRelics = bonusRelics(data, session)
		data.Progress.Clears++
		data.Progress.BossClears[session.Dungeon]++
		if seals := eventSeals(session); seals > 0 {
			data.Progress.Materials["seal"] += seals
			result.MaterialsGain = map[string]int{"seal": seals}
			result.Message = fmt.Sprintf("%s: аномалия зачищена! +%d ауры, +%d опыта, печати ×%d", session.Event.Name, gain, xp, seals)
		}
		d, _ := dungeonByID(session.Dungeon)
		dr := difficultyRank(session.Difficulty)
		if d.Level > data.Progress.BestDungeon || (d.Level == data.Progress.BestDungeon && (dr > data.Progress.BestDifficulty || (dr == data.Progress.BestDifficulty && session.Errors < data.Progress.BestErrors))) {
			data.Progress.BestDungeon = d.Level
			data.Progress.BestDifficulty = dr
			data.Progress.BestErrors = session.Errors
		}
	}
	if session.Kind == SessionTower {
		data.Progress.TowerFloor = max(data.Progress.TowerFloor, session.Floor)
	}
	return result
}
func (s *Service) reward(data *row, session *Session, now time.Time) (int64, int, *Relic) {
	if session.BossHP > 0 {
		return 0, 0, nil
	}
	if session.Kind == SessionGrind {
		m := grindMultiplier(max(0, data.GrindToday-1))
		return int64(25 * m), int(7 * m), nil
	}
	gain := int64(60 + session.Correct*12)
	xp := 25 + session.Correct*5
	if session.Kind == SessionGate {
		d, _ := dungeonByID(session.Dungeon)
		gain += int64(d.Level * 20)
		xp += d.Level * 4
		if acceptedEvent(session) {
			gain += int64(80 * eventSeals(session))
			xp += 20 * eventSeals(session)
		}
	}
	if session.Kind == SessionTower {
		gain += int64(session.Floor * 15)
		xp += session.Floor * 3
	}
	return gain, xp, rollLoot(s.rnd, session, &data.Progress)
}

// grantRelic stores a relic, awakening it instead when the player already owns
// it. A duplicate is never wasted: it raises the level, and past level five it
// pays aura.
func (s *Service) grantRelic(ctx context.Context, tx pgx.Tx, userID int64, relic Relic) (int64, error) {
	var level int
	err := tx.QueryRow(ctx,
		`SELECT level FROM aura_relics WHERE user_id = $1 AND relic_key = $2 FOR UPDATE`,
		userID, relic.Key).Scan(&level)
	if err == pgx.ErrNoRows {
		_, err = tx.Exec(ctx,
			`INSERT INTO aura_relics (user_id, relic_key, rank) VALUES ($1, $2, $3)`,
			userID, relic.Key, relic.Rank)
		return 0, err
	}
	if err != nil {
		return 0, err
	}
	if level < 5 {
		_, err = tx.Exec(ctx,
			`UPDATE aura_relics SET level = level + 1 WHERE user_id = $1 AND relic_key = $2`,
			userID, relic.Key)
		return 0, err
	}
	return 200, nil
}

// checkRequest rejects retries under the player lock before this record is written.
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
	var rawSession, rawProgress []byte
	var grindDay, lastSeen *time.Time
	err := tx.QueryRow(ctx,
		`SELECT aura, total_aura, level, xp, stat_points,
		        stat_str, stat_agi, stat_int, stat_vit, stat_per,
		        hero_class, farm_ready_at, arena_ready_at, gate_ready_at, tower_ready_at,
		        grind_day, grind_today, session, streak, last_seen_day, progress
		   FROM aura_players WHERE user_id = $1 FOR UPDATE`, userID,
	).Scan(&out.Aura, &out.TotalAura, &out.Level, &out.XP, &out.StatPoints,
		&out.Stats.Str, &out.Stats.Agi, &out.Stats.Int, &out.Stats.Vit, &out.Stats.Per,
		&out.HeroClass, &out.FarmReadyAt, &out.ArenaReadyAt, &out.GateReadyAt, &out.TowerReadyAt,
		&grindDay, &out.GrindToday, &rawSession, &out.Streak, &lastSeen, &rawProgress)
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
	if err := json.Unmarshal(rawProgress, &out.Progress); err != nil {
		return row{}, err
	}
	normalizeProgress(&out, s.now())
	normalizeLegacySession(&out, userID)
	return out, nil
}

// wornRelicsTx is wornRelics inside a transaction.
func (s *Service) wornRelicsTx(ctx context.Context, tx pgx.Tx, userID int64) ([]Relic, error) {
	rows, err := tx.Query(ctx,
		`SELECT relic_key, level FROM aura_relics WHERE user_id = $1 AND slot <> ''`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Relic, 0, 3)
	for rows.Next() {
		var key string
		var level int
		if err := rows.Scan(&key, &level); err != nil {
			return nil, err
		}
		if relic, ok := relicByKey(key); ok {
			out = append(out, scaledRelic(relic, level))
		}
	}
	return out, rows.Err()
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
