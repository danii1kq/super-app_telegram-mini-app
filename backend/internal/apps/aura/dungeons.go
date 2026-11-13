package aura

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math"
	"time"

	"github.com/jackc/pgx/v5"
)

type Progress struct {
	SSEventPity    int             `json:"ss_event_pity"`
	SSSEventPity   int             `json:"sss_event_pity"`
	Difficulty     string          `json:"difficulty"`
	Materials      map[string]int  `json:"materials"`
	Rank           string          `json:"rank"`
	Clears         int             `json:"clears"`
	TowerFloor     int             `json:"tower_floor"`
	Correct        int             `json:"correct"`
	BossClears     map[string]int  `json:"boss_clears"`
	Claimed        map[string]bool `json:"claimed"`
	BestDungeon    int             `json:"best_dungeon"`
	BestDifficulty int             `json:"best_difficulty"`
	BestErrors     int             `json:"best_errors"`
	Pity           int             `json:"pity"`
}

type Dungeon struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Level       int    `json:"level"`
	Material    string `json:"material"`
	Boss        string `json:"boss"`
	Theme       string `json:"theme"`
}

var dungeons = []Dungeon{
	{"crypt", "Затопленный склеп", "Пройди затопленные залы и останови хранителя печати.", 1, "bone", "Хранитель склепа", "crypt"},
	{"forge", "Пепельная кузница", "Закали свою сборку в огне древней кузницы.", 5, "ember", "Пепельный колосс", "forge"},
	{"void", "Разлом пустоты", "Найди путь среди искажённых рун и победи владыку разлома.", 10, "void", "Владыка пустоты", "void"},
}

func dungeonByID(id string) (Dungeon, bool) {
	for _, d := range dungeons {
		if d.ID == id {
			return d, true
		}
	}
	return Dungeon{}, false
}
func normalizeProgress(data *row, now time.Time) {
	p := &data.Progress
	if p.Difficulty == "" {
		p.Difficulty = "easy"
	}
	if p.Rank == "" {
		p.Rank = "E"
	}
	if p.Materials == nil {
		p.Materials = map[string]int{}
	}
	if p.BossClears == nil {
		p.BossClears = map[string]int{}
	}
	if p.Claimed == nil {
		p.Claimed = map[string]bool{}
	}
	if data.GrindDay == nil || !sameDay(*data.GrindDay, now) {
		today := now.UTC().Truncate(24 * time.Hour)
		data.GrindDay = &today
		data.GrindToday = 0
	}
}
func heroMaxHP(stats Stats) int { return 100 + stats.Vit*12 }
func difficultyLevel(difficulty string) int {
	switch difficulty {
	case "normal":
		return 12
	case "hard":
		return 30
	}
	return 1
}
func difficultyRank(difficulty string) int {
	switch difficulty {
	case "normal":
		return 2
	case "hard":
		return 3
	}
	return 1
}
func validDifficulty(d string) bool { return d == "easy" || d == "normal" || d == "hard" }
func newID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}
func normalizeLegacySession(data *row, userID int64) {
	session := data.Session
	if session == nil || session.ID != "" {
		return
	}
	session.ID = fmt.Sprintf("legacy-%d-%d", userID, session.AskedAt.UnixNano())
	session.QuestionID = session.ID + "-question"
	session.HeroMaxHP = heroMaxHP(data.Stats)
	session.HeroHP = session.HeroMaxHP
	session.HeroStats = data.Stats
	session.HeroPath = data.HeroClass
	session.Phase = "combat"
	session.Room = 1
	session.Rooms = 1
	session.Difficulty = data.Progress.Difficulty
	session.QuestionDeadline = session.AskedAt.Add(time.Duration(max(20, session.Seconds)) * time.Second)
	session.Intent = "Удар при ошибке"
}
func (s *Service) nextQuestion(session *Session, now time.Time) error {
	id, err := newID()
	if err != nil {
		return err
	}
	session.QuestionID = id
	session.Question = Generate(s.rnd, difficultyLevel(session.Difficulty), "")
	session.Seconds = secondsForQuestion(difficultyLevel(session.Difficulty), session.HeroPath) + min(10, session.HeroStats.Agi/2) + min(5, session.HeroStats.Per/3)
	if !session.TimeUsed && relicBonus(session.Worn, "time") > 0 {
		session.Seconds += int(relicBonus(session.Worn, "time"))
		session.TimeUsed = true
	}
	session.AskedAt = now
	session.QuestionDeadline = now.Add(time.Duration(session.Seconds) * time.Second)
	session.ExpiresAt = now.Add(SessionTTL)
	session.Intent = "Удар при ошибке"
	if session.Round%3 == 0 {
		session.Intent = "Сокрушение: реши пример, чтобы закрыться"
	}
	if session.Round%4 == 0 {
		session.Intent = "Стойка защиты: урон снижен"
	}
	if session.Room == session.Rooms && session.Kind == SessionGate && !acceptedEvent(session) {
		switch session.Dungeon {
		case "crypt":
			if session.Round%2 == 0 {
				session.Intent = "Костяная броня: каждый третий удар пробивает защиту"
			}
		case "forge":
			if session.Round%2 == 0 {
				session.Intent = "Размах колосса: ошибка наносит двойной урон"
			}
		case "void":
			session.Intent = "Пустота: ошибка лечит босса и отнимает заряд"
		}
	}
	setEventIntent(session)
	return nil
}
func (s *Service) enterRoom(session *Session, route string, now time.Time) error {
	session.Phase = "combat"
	session.Elite = route == "elite" || route == "chest"
	base := 28 + session.Room*10
	if session.Room == session.Rooms {
		base += 25
	}
	if d, ok := dungeonByID(session.Dungeon); ok {
		base += (d.Level - 1) * 6
		session.BossName = []string{"", "Страж затопленных залов", "Рунный дозорный", d.Boss}[min(session.Room, 3)]
	}
	if session.Kind == SessionTower {
		base += session.Floor * 5
		session.BossName = fmt.Sprintf("Страж этажа %d", session.Floor)
	}
	if session.Kind == SessionArena {
		session.BossName = "Хранитель арены"
	}
	if session.Elite {
		base = base * 3 / 2
		session.BossName = "Элита: " + session.BossName
	}
	session.BossHP = base
	session.BossMaxHP = base
	configureEventEnemy(session)
	return s.nextQuestion(session, now)
}
func calculateHit(session *Session, rndFloat float64) int {
	damage := 18 + session.HeroStats.Str*3 + session.Combo*2
	damage += int(float64(damage) * (relicBonus(session.Worn, "damage")))
	if session.Kind == SessionArena {
		damage += int(float64(damage) * bonusFor(session.HeroPath).ArenaDamage)
	}
	if session.Kind == SessionGate || session.Kind == SessionTower {
		damage += int(float64(damage) * bonusFor(session.HeroPath).Raid)
	}
	if rndFloat < math.Min(.75, critChance(session.HeroStats, session.Worn)+bonusFor(session.HeroPath).Crit) {
		damage *= 2
	}
	if relicBonus(session.Worn, "echo") > 0 && session.Combo%3 == 0 {
		damage += int(float64(damage) * relicBonus(session.Worn, "echo"))
	}
	if relicBonus(session.Worn, "execute") > 0 && session.BossHP*100 <= session.BossMaxHP*35 {
		damage += int(float64(damage) * relicBonus(session.Worn, "execute"))
	}
	if session.Round%4 == 0 && !acceptedEvent(session) {
		damage = max(1, damage*3/4)
	}
	if !acceptedEvent(session) && session.Dungeon == "crypt" && session.Room == session.Rooms && session.Round%2 == 0 && session.Combo%3 != 0 {
		damage = max(1, damage/2)
	}
	if session.Empowered {
		damage *= 2
		session.Empowered = false
	}
	return eventHit(session, damage)
}
func incomingHit(session *Session, evadeRoll float64) int {
	if session.Shield {
		session.Shield = false
		return 0
	}
	if evadeRoll < math.Min(.3, float64(session.HeroStats.Agi)*.015) {
		return 0
	}
	damage := 18 + session.Room*3
	if d, ok := dungeonByID(session.Dungeon); ok {
		damage += d.Level
	}
	if session.Round%3 == 0 {
		damage = damage * 3 / 2
	}
	if !acceptedEvent(session) && session.Dungeon == "forge" && session.Room == session.Rooms && session.Round%2 == 0 {
		damage *= 2
	}
	if session.Elite {
		damage += 5
	}
	damage = eventIncoming(session, damage)
	reduction := math.Min(.6, relicBonus(session.Worn, "negate")+bonusFor(session.HeroPath).Negate+float64(session.HeroStats.Vit)*.005)
	return max(1, int(float64(damage)*(1-reduction)))
}
func (s *Service) checkRequest(ctx context.Context, tx pgx.Tx, userID int64, requestID string) error {
	if requestID == "" || len(requestID) > 128 {
		return bad("Некорректный номер запроса")
	}
	var seen bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM aura_runs WHERE user_id=$1 AND request_id=$2)`, userID, requestID).Scan(&seen); err != nil {
		return err
	}
	if seen {
		return bad("Этот запрос уже обработан. Обнови состояние")
	}
	return nil
}
func (s *Service) resultPlayer(ctx context.Context, userID int64, result Result, err error) (Result, error) {
	if err != nil {
		return Result{}, err
	}
	player, err := s.snapshot(ctx, userID)
	result.Player = player
	return result, err
}

func auraWeekStart(now time.Time) time.Time {
	now = now.UTC()
	day := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	return day.AddDate(0, 0, -(int(day.Weekday())+6)%7)
}
func (s *Service) recordWeekly(ctx context.Context, tx pgx.Tx, userID int64, session *Session, now time.Time) error {
	dungeon, _ := dungeonByID(session.Dungeon)
	_, err := tx.Exec(ctx, `INSERT INTO aura_weekly_scores(week,user_id,depth,difficulty,errors,clears) VALUES($1,$2,$3,$4,$5,1)
 ON CONFLICT(week,user_id) DO UPDATE SET
 depth=CASE WHEN (EXCLUDED.depth,EXCLUDED.difficulty,-EXCLUDED.errors)>(aura_weekly_scores.depth,aura_weekly_scores.difficulty,-aura_weekly_scores.errors) THEN EXCLUDED.depth ELSE aura_weekly_scores.depth END,
 difficulty=CASE WHEN (EXCLUDED.depth,EXCLUDED.difficulty,-EXCLUDED.errors)>(aura_weekly_scores.depth,aura_weekly_scores.difficulty,-aura_weekly_scores.errors) THEN EXCLUDED.difficulty ELSE aura_weekly_scores.difficulty END,
 errors=CASE WHEN (EXCLUDED.depth,EXCLUDED.difficulty,-EXCLUDED.errors)>(aura_weekly_scores.depth,aura_weekly_scores.difficulty,-aura_weekly_scores.errors) THEN EXCLUDED.errors ELSE aura_weekly_scores.errors END,
 clears=aura_weekly_scores.clears+1`, auraWeekStart(now), userID, dungeon.Level, difficultyRank(session.Difficulty), session.Errors)
	return err
}
