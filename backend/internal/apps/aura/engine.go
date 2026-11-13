package aura

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"time"
)

// Tuning. Every number here is a design decision, so they live together rather
// than scattered through the handlers.
const (
	// Meditation is the slow loop; successful combat has a short recovery.
	FarmCooldown  = 4 * time.Hour
	ArenaCooldown = 10 * time.Minute
	GateCooldown  = 20 * time.Minute
	TowerCooldown = 10 * time.Minute

	// A session expires if the player walks away mid-run. Without this the row
	// would hold a dead challenge forever.
	SessionTTL = 10 * time.Minute

	// A generous hard limit keeps any stalled run bounded.
	CombatRounds = 80

	// Grind: unlimited starts, but the reward decays across the day.
	GrindDailyCap = 50

	// The tenth rewarded relic is at least A after nine lower-rank drops.
	RelicPity = 10

	// Cost of rerolling the hero path.
	ClassRerollCost = 300
)

// SessionKind is what the open challenge is.
type SessionKind string

const (
	SessionArena SessionKind = "arena"
	SessionGate  SessionKind = "gate"
	SessionGrind SessionKind = "grind"
	SessionTower SessionKind = "tower"
)

// Session is the open challenge, stored in the player row.
type Session struct {
	HealingUsed       bool          `json:"healing_used"`
	EmpoweredFinisher bool          `json:"empowered_finisher"`
	Sets              []string      `json:"sets"`
	PendingEvent      *DungeonEvent `json:"pending_event,omitempty"`
	Event             *DungeonEvent `json:"event,omitempty"`
	ID                string        `json:"id"`
	QuestionID        string        `json:"question_id"`
	QuestionDeadline  time.Time     `json:"question_deadline"`
	HeroHP            int           `json:"hero_hp"`
	HeroMaxHP         int           `json:"hero_max_hp"`
	Room              int           `json:"room"`
	Rooms             int           `json:"rooms"`
	Phase             string        `json:"phase"`
	Dungeon           string        `json:"dungeon"`
	Intent            string        `json:"intent"`
	Charge            int           `json:"charge"`
	AbilityReady      bool          `json:"ability_ready"`
	Difficulty        string        `json:"difficulty"`
	Errors            int           `json:"errors"`
	Elite             bool          `json:"elite"`
	Shield            bool          `json:"shield"`
	Empowered         bool          `json:"empowered"`
	DoubleReward      bool          `json:"double_reward"`
	TimeUsed          bool          `json:"time_used"`
	Worn              []Relic       `json:"worn_snapshot,omitempty"`
	HeroStats         Stats         `json:"hero_stats"`
	HeroPath          HeroClass     `json:"hero_path"`
	Kind              SessionKind   `json:"kind"`
	Question          Question      `json:"question"`
	AskedAt           time.Time     `json:"asked_at"`
	ExpiresAt         time.Time     `json:"expires_at"`
	Round             int           `json:"round"`
	Rounds            int           `json:"rounds"`
	Correct           int           `json:"correct"`
	Combo             int           `json:"combo"`
	BestCombo         int           `json:"best_combo"`
	Damage            int           `json:"damage"`
	BossHP            int           `json:"boss_hp"`
	BossMaxHP         int           `json:"boss_max_hp"`
	BossName          string        `json:"boss_name"`
	Floor             int           `json:"floor"`
	Pity              int           `json:"pity"`
	Ability           bool          `json:"ability_used"`
	Seconds           int           `json:"seconds"`
	RequestIDs        []string      `json:"request_ids"`
}

// Player is the hero as the client sees it.
type Player struct {
	MaxHP        int            `json:"max_hp"`
	Power        int            `json:"power"`
	Rank         string         `json:"rank"`
	Difficulty   string         `json:"difficulty"`
	Materials    map[string]int `json:"materials"`
	Clears       int            `json:"clears"`
	TowerFloor   int            `json:"tower_floor"`
	Aura         int64          `json:"aura"`
	TotalAura    int64          `json:"total_aura"`
	Level        int            `json:"level"`
	XP           int            `json:"xp"`
	XPToNext     int            `json:"xp_to_next"`
	StatPoints   int            `json:"stat_points"`
	Stats        Stats          `json:"stats"`
	HeroClass    HeroClass      `json:"hero_class"`
	ClassName    string         `json:"class_name"`
	Ability      any            `json:"ability"`
	FarmReadyAt  *time.Time     `json:"farm_ready_at"`
	ArenaReadyAt *time.Time     `json:"arena_ready_at"`
	GateReadyAt  *time.Time     `json:"gate_ready_at"`
	TowerReadyAt *time.Time     `json:"tower_ready_at"`
	GrindToday   int            `json:"grind_today"`
	GrindCap     int            `json:"grind_cap"`
	Streak       int            `json:"streak"`
	Session      *Session       `json:"session"`
}

// Stats are the five attributes a level point can go into.
type Stats struct {
	Str int `json:"str"`
	Agi int `json:"agi"`
	Int int `json:"int"`
	Vit int `json:"vit"`
	Per int `json:"per"`
}

// xpForLevel is the curve. It grows linearly, which keeps the early levels
// quick and the late ones meaningful without a wall.
func xpForLevel(level int) int {
	if level < 1 {
		level = 1
	}
	return 80 + (level-1)*40
}

// levelForXP walks the curve. A cap keeps the arithmetic bounded.
func levelForXP(xp int) int {
	const maxLevel = 100
	level := 1
	remaining := xp
	for level < maxLevel && remaining >= xpForLevel(level) {
		remaining -= xpForLevel(level)
		level++
	}
	return level
}

// grindMultiplier decays the reward across the daily cap. The first fifteen
// answers pay in full, then half, then a fifth, then nothing: a player can keep
// playing but cannot farm the same channel forever.
func grindMultiplier(today int) float64 {
	switch {
	case today < 15:
		return 1
	case today < 30:
		return 0.5
	case today < GrindDailyCap:
		return 0.2
	}
	return 0
}

// arenaBoss picks the opponent and its health for a rank. The name is flavour,
// the health is the difficulty curve.
func arenaBoss(rnd *rand.Rand, level int) (string, int) {
	names := []string{"Лесной гоблин", "Каменный страж", "Пепельный жрец", "Ледяная ведьма", "Владыка конца"}
	index := 0
	switch {
	case level >= 40:
		index = 4
	case level >= 25:
		index = 3
	case level >= 15:
		index = 2
	case level >= 7:
		index = 1
	}
	return names[index], 60 + level*20
}

// secondsForQuestion is the time budget. The class bonus shifts it, and the
// floor keeps a berserker from getting an impossible clock.
func secondsForQuestion(level int, class HeroClass) int {
	base := 20 + level/3
	if base > 40 {
		base = 40
	}
	base += bonusFor(class).Time
	if base < 12 {
		base = 12
	}
	return base
}

// critChance is the chance of a doubled hit, from intelligence and relics.
func critChance(stats Stats, relics []Relic) float64 {
	chance := float64(stats.Int) * 0.01
	for _, relic := range relics {
		if relic.Passive == "crit" {
			chance += relic.Value
		}
	}
	if chance > 0.75 {
		chance = 0.75
	}
	return chance
}

// relicBonus sums one passive across the worn relics.
func relicBonus(relics []Relic, passive string) float64 {
	total := 0.0
	for _, relic := range relics {
		if relic.Passive == passive {
			total += relic.Value
		}
	}
	return total
}

// sessionJSON marshals a session for the jsonb column.
func sessionJSON(session *Session) ([]byte, error) {
	if session == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(session)
}

// sessionFromJSON reads the column back. An empty object means no session.
func sessionFromJSON(raw []byte) (*Session, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var probe map[string]any
	if err := json.Unmarshal(raw, &probe); err != nil {
		return nil, err
	}
	if len(probe) == 0 {
		return nil, nil
	}
	var session Session
	if err := json.Unmarshal(raw, &session); err != nil {
		return nil, err
	}
	return &session, nil
}

// expireSession drops a session whose clock ran out. The caller decides what
// that costs: an expired arena is a lost attempt, an expired grind is nothing.
func expireSession(session *Session, now time.Time) (*Session, bool) {
	if session == nil {
		return nil, false
	}
	if now.After(session.ExpiresAt) {
		return nil, true
	}
	return session, false
}

// describeSession is the one-line status shown above the question.
func describeSession(session *Session) string {
	switch session.Kind {
	case SessionArena:
		return fmt.Sprintf("Арена: раунд %d из %d, комбо %d", session.Round, session.Rounds, session.Combo)
	case SessionGate:
		return fmt.Sprintf("Подземелье: %s, %d из %d HP", session.BossName, session.BossHP, session.BossMaxHP)
	case SessionTower:
		return fmt.Sprintf("Башня: этаж %d, страж %d из %d HP", session.Floor, session.BossHP, session.BossMaxHP)
	case SessionGrind:
		return fmt.Sprintf("Вылазка: комбо %d", session.Combo)
	}
	return ""
}
