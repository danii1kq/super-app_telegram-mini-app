package aura

import (
	"context"
	"errors"
	"math/rand"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/danii1kq/srunal/internal/platform/httpapi"
	"github.com/danii1kq/srunal/internal/platform/postgres"
)

// Service owns the hero rows and the challenge engine.
type Service struct {
	pool *postgres.Pool
	// rnd is injectable so tests are deterministic.
	rnd *rand.Rand
	now func() time.Time
}

func NewService(pool *postgres.Pool) *Service {
	return &Service{
		pool: pool,
		rnd:  rand.New(rand.NewSource(time.Now().UnixNano())),
		now:  func() time.Time { return time.Now().UTC() },
	}
}

// row is the database shape of a hero.
type row struct {
	Aura         int64
	TotalAura    int64
	Level        int
	XP           int
	StatPoints   int
	Stats        Stats
	HeroClass    HeroClass
	FarmReadyAt  *time.Time
	ArenaReadyAt *time.Time
	GateReadyAt  *time.Time
	TowerReadyAt *time.Time
	GrindDay     *time.Time
	GrindToday   int
	Session      *Session
	Streak       int
	LastSeenDay  *time.Time
}

// load reads the hero, creating the row on first visit.
//
// The insert is ON CONFLICT DO NOTHING so two simultaneous first requests
// cannot fail: the second one simply reads what the first wrote.
func (s *Service) load(ctx context.Context, userID int64) (row, error) {
	if _, err := s.pool.Exec(ctx,
		`INSERT INTO aura_players (user_id) VALUES ($1) ON CONFLICT DO NOTHING`, userID,
	); err != nil {
		return row{}, err
	}

	var out row
	var rawSession []byte
	var grindDay, lastSeen *time.Time
	err := s.pool.QueryRow(ctx,
		`SELECT aura, total_aura, level, xp, stat_points,
		        stat_str, stat_agi, stat_int, stat_vit, stat_per,
		        hero_class, farm_ready_at, arena_ready_at, gate_ready_at, tower_ready_at,
		        grind_day, grind_today, session, streak, last_seen_day
		   FROM aura_players WHERE user_id = $1`, userID,
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

// save writes the hero back. Every mutation goes through here so the updated_at
// stamp is never forgotten.
func (s *Service) save(ctx context.Context, tx pgx.Tx, userID int64, data row) error {
	raw, err := sessionJSON(data.Session)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx,
		`UPDATE aura_players
		    SET aura = $2, total_aura = $3, level = $4, xp = $5, stat_points = $6,
		        stat_str = $7, stat_agi = $8, stat_int = $9, stat_vit = $10, stat_per = $11,
		        hero_class = $12, farm_ready_at = $13, arena_ready_at = $14,
		        gate_ready_at = $15, tower_ready_at = $16, grind_day = $17, grind_today = $18,
		        session = $19, streak = $20, last_seen_day = $21, updated_at = now()
		  WHERE user_id = $1`,
		userID, data.Aura, data.TotalAura, data.Level, data.XP, data.StatPoints,
		data.Stats.Str, data.Stats.Agi, data.Stats.Int, data.Stats.Vit, data.Stats.Per,
		data.HeroClass, data.FarmReadyAt, data.ArenaReadyAt,
		data.GateReadyAt, data.TowerReadyAt, data.GrindDay, data.GrindToday,
		raw, data.Streak, data.LastSeenDay)
	return err
}

// wornRelics returns only the equipped relics, because an unworn relic does
// nothing by design.
func (s *Service) wornRelics(ctx context.Context, userID int64) ([]Relic, error) {
	rows, err := s.pool.Query(ctx,
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

// snapshot builds the client view.
func (s *Service) snapshot(ctx context.Context, userID int64) (Player, error) {
	data, err := s.load(ctx, userID)
	if err != nil {
		return Player{}, err
	}
	now := s.now()

	// An expired session is dropped here, so the client never sees a dead one.
	if data.Session != nil && now.After(data.Session.ExpiresAt) {
		data.Session = nil
	}
	// The answer stays in the row so a restart mid-run does not break the
	// question, but the client must never see it: it would be a free win.
	if data.Session != nil {
		view := *data.Session
		view.Question.Answer = 0
		data.Session = &view
	}

	player := Player{
		Aura:         data.Aura,
		TotalAura:    data.TotalAura,
		Level:        data.Level,
		XP:           data.XP,
		XPToNext:     xpForLevel(data.Level),
		StatPoints:   data.StatPoints,
		Stats:        data.Stats,
		HeroClass:    data.HeroClass,
		FarmReadyAt:  data.FarmReadyAt,
		ArenaReadyAt: data.ArenaReadyAt,
		GateReadyAt:  data.GateReadyAt,
		TowerReadyAt: data.TowerReadyAt,
		GrindToday:   data.GrindToday,
		GrindCap:     GrindDailyCap,
		Streak:       data.Streak,
		Session:      data.Session,
	}
	for _, entry := range classCatalog {
		if entry.ID == data.HeroClass {
			player.ClassName = entry.Name
		}
	}
	if ability, ok := abilities[data.HeroClass]; ok {
		player.Ability = ability
	}
	return player, nil
}

// bad is the shorthand for a validation error.
func bad(message string) error {
	return httpapi.BadRequest("aura", message)
}

// requireReady fails when a cooldown has not elapsed.
func requireReady(readyAt *time.Time, now time.Time, label string) error {
	if readyAt != nil && now.Before(*readyAt) {
		return bad(label + " ещё не готова")
	}
	return nil
}

// grantXP adds experience and levels the hero up, returning the new level and
// the points it earned. A level-up is worth one stat point.
func grantXP(data *row, amount int) (int, int) {
	data.XP += amount
	newLevel := levelForXP(data.XP)
	gained := 0
	for data.Level < newLevel {
		data.Level++
		data.StatPoints++
		gained++
	}
	return data.Level, gained
}

// errNoSession is returned when a submit arrives with nothing open.
var errNoSession = errors.New("aura: no open session")

// rollRequestID is the idempotency key for one submit. The client sends it, and
// a repeat is a no-op rather than a second reward.
func (s *Service) seenRequest(ctx context.Context, userID int64, requestID string) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM aura_runs WHERE user_id = $1 AND request_id = $2)`,
		userID, requestID).Scan(&exists)
	return exists, err
}
