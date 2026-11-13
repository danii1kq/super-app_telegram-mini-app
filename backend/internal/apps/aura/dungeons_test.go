package aura

import (
	"testing"
	"time"
)

func testRow() row { data := row{Level: 1}; normalizeProgress(&data, time.Now()); return data }
func TestFailedCombatNeverGrantsLoot(t *testing.T) {
	s := &Service{rnd: seeded(5)}
	for _, kind := range []SessionKind{SessionGate, SessionArena, SessionTower, SessionGrind} {
		data := testRow()
		data.Progress.Materials["bone"] = 7
		session := &Session{Kind: kind, Dungeon: "crypt", BossHP: 40, HeroHP: 0, Correct: 0, Room: 3, Rooms: 3}
		data.Session = session
		result := s.finishRun(&data, session, false, time.Now())
		if result.Won || result.Relic != nil || result.AuraGain != 0 || data.Aura != 0 || data.XP != 0 || data.Progress.Clears != 0 || data.Progress.Materials["bone"] != 7 || data.Session != nil {
			t.Fatalf("failedrun paid or lost bank: %s %+v", kind, result)
		}
	}
}
func TestRelicAwakeningScalesEffectAndPreservesCatalog(t *testing.T) {
	base, _ := relicByKey("rusty_blade")
	one, five := scaledRelic(base, 1), scaledRelic(base, 5)
	if five.Value <= one.Value || five.UpgradeCost != 0 || one.UpgradeCost <= 0 {
		t.Fatal("awakening has no effect")
	}
	unchanged, _ := relicByKey("rusty_blade")
	if unchanged.Value != base.Value {
		t.Fatal("catalog mutated")
	}
	for _, key := range []string{"echo_blade", "troll_heart", "hourglass", "executioner"} {
		item, ok := relicByKey(key)
		if !ok {
			t.Fatal(key)
		}
		view := scaledRelic(item, 1)
		if view.Description == "" || view.Effect == "" || view.CraftAura <= 0 || len(view.CraftCost) == 0 {
			t.Fatal("missing signature effect or recipe", key)
		}
	}
}
func TestBossMechanicsAndStatsAffectCombat(t *testing.T) {
	s := &Service{rnd: seeded(3)}
	now := time.Now()
	crypt := Session{Kind: SessionGate, Dungeon: "crypt", Room: 3, Rooms: 3, Round: 2, Combo: 2, HeroHP: 100, HeroMaxHP: 100, Difficulty: "easy"}
	if err := s.nextQuestion(&crypt, now); err != nil {
		t.Fatal(err)
	}
	armored := calculateHit(&crypt, 1)
	crypt.Combo = 3
	broken := calculateHit(&crypt, 1)
	if broken < armored*2 {
		t.Fatal("third strike did not break cryptarmor")
	}
	normal := Session{Kind: SessionGate, Dungeon: "crypt", Room: 3, Rooms: 3, Round: 2, HeroHP: 100, HeroMaxHP: 100}
	forge := normal
	forge.Dungeon = "forge"
	if incomingHit(&forge, 1) <= incomingHit(&normal, 1) {
		t.Fatal("forgeheavy attack not stronger")
	}
	protected := normal
	protected.HeroStats.Vit = 30
	if incomingHit(&protected, 1) >= incomingHit(&normal, 1) || heroMaxHP(protected.HeroStats) <= heroMaxHP(normal.HeroStats) {
		t.Fatal("vitality did not work")
	}
	agile := normal
	agile.HeroStats.Agi = 20
	if incomingHit(&agile, .1) != 0 {
		t.Fatal("agility did not dodge")
	}
	before := calculateHit(&normal, 1)
	normal.HeroStats.Str = 5
	if calculateHit(&normal, 1) <= before {
		t.Fatal("strength did not work")
	}
}
func TestQuestionClockAndDifficultyAreServerOwned(t *testing.T) {
	s := &Service{rnd: seeded(9)}
	now := time.Now()
	easy := Session{Difficulty: "easy", HeroPath: ClassStrategist, Round: 1}
	hard := Session{Difficulty: "hard", HeroStats: Stats{Agi: 10, Per: 9}, Round: 1}
	if err := s.nextQuestion(&easy, now); err != nil {
		t.Fatal(err)
	}
	if err := s.nextQuestion(&hard, now); err != nil {
		t.Fatal(err)
	}
	if easy.QuestionID == "" || easy.QuestionID == hard.QuestionID || !easy.QuestionDeadline.After(now) || easy.Question.Topic == TopicEquation || easy.Question.Topic == TopicSqrt {
		t.Fatal("bad question snapshots")
	}
	id := easy.QuestionID
	if err := s.nextQuestion(&easy, now); err != nil {
		t.Fatal(err)
	}
	if easy.QuestionID == id {
		t.Fatal("reusedquestionID")
	}
	ordinary := Session{Difficulty: "easy", Round: 1}
	perceptive := ordinary
	perceptive.HeroStats.Per = 9
	_ = s.nextQuestion(&ordinary, now)
	_ = s.nextQuestion(&perceptive, now)
	if perceptive.Seconds <= ordinary.Seconds {
		t.Fatal("perception does not affect time")
	}
}
func TestDailyGrindResetAndQuestClaimFlags(t *testing.T) {
	now := time.Date(2026, 11, 13, 0, 0, 0, 0, time.UTC)
	yesterday := now.Add(-time.Hour)
	data := row{Level: 5, GrindDay: &yesterday, GrindToday: 50, Progress: Progress{Clears: 3, Correct: 30, BossClears: map[string]int{"crypt": 3}, Claimed: map[string]bool{"first_clear": true}}}
	normalizeProgress(&data, now)
	if data.GrindToday != 0 {
		t.Fatal("dailygrind never resets")
	}
	data.GrindToday = 7
	normalizeProgress(&data, now.Add(time.Hour))
	if data.GrindToday != 7 {
		t.Fatal("same day reset twice")
	}
	for _, q := range questsFor(data) {
		if q.ID == "first_clear" && !q.Claimed {
			t.Fatal("claimed flag lost")
		}
		if q.ID == "rank_d" && q.Progress != q.Goal {
			t.Fatal("ranktrial not achievable")
		}
	}
}
func TestLegacySessionNormalizationKeepsQuestionAndAddsCombatState(t *testing.T) {
	data := testRow()
	data.Session = &Session{Kind: SessionGate, Question: Question{Text: "2 + 2", Answer: 4}, AskedAt: time.Now(), Seconds: 20, BossHP: 60, Round: 2}
	normalizeLegacySession(&data, 7)
	id := data.Session.QuestionID
	if id == "" || data.Session.Question.Answer != 4 || data.Session.HeroHP != 100 || data.Session.BossHP != 60 {
		t.Fatal("legacyrun damaged")
	}
	normalizeLegacySession(&data, 7)
	if data.Session.QuestionID != id {
		t.Fatal("legacyID notstable")
	}
}
func TestWeeklyBoundaryUsesMondayUTC(t *testing.T) {
	for day := 9; day <= 15; day++ {
		week := auraWeekStart(time.Date(2026, 11, day, 23, 59, 0, 0, time.UTC))
		if week.Day() != 9 || week.Hour() != 0 {
			t.Fatal("wrongweeklyboundary", week)
		}
	}
}
