package aura

import (
	"encoding/json"
	"testing"
	"time"
)

func TestEventPlanProbabilitiesAndPersistence(t *testing.T) {
	counts := map[string]int{}
	for roll := 0; roll < 1000; roll++ {
		event := plannedEvent(roll, 2)
		if event != nil {
			counts[event.ID]++
			if event.Accepted || event.RevealAfterRoom != 2 {
				t.Fatal("invalid pending event")
			}
		}
	}
	if counts["false_gate"] != 8 || counts["red_gate"] != 12 || counts["hidden_shrine"] != 15 {
		t.Fatal("event probability drift", counts)
	}
	session := Session{Kind: SessionGate, Room: 1, Rooms: 3, Phase: "combat", PendingEvent: plannedEvent(0, 2)}
	data, err := json.Marshal(session)
	if err != nil {
		t.Fatal(err)
	}
	var restored Session
	if err = json.Unmarshal(data, &restored); err != nil {
		t.Fatal(err)
	}
	if revealDungeonEvent(&restored, time.Now()) {
		t.Fatal("revealed before selected room")
	}
	restored.Room = 2
	if !revealDungeonEvent(&restored, time.Now()) || restored.Phase != "event" || restored.Event.Rank != "S" || restored.PendingEvent != nil {
		t.Fatal("did not reveal saved event")
	}
	if revealDungeonEvent(&restored, time.Now()) {
		t.Fatal("event revealed twice")
	}
	if restored.Event.Accepted {
		t.Fatal("reveal silently accepted higher danger")
	}
}
func TestAcceptedEventsHaveRealDangerAndKeepMathDifficulty(t *testing.T) {
	s := &Service{rnd: seeded(6)}
	now := time.Now()
	for _, roll := range []int{0, 8, 20} {
		event := plannedEvent(roll, 1)
		event.RevealAfterRoom = 0
		session := Session{Kind: SessionGate, Dungeon: "crypt", Room: 1, Rooms: 3, Phase: "event", Event: event, HeroHP: 100, HeroMaxHP: 100, Difficulty: "easy", Round: 4}
		if err := s.enterDungeonEvent(&session, now); err != nil {
			t.Fatal(err)
		}
		if !session.Event.Accepted || session.Phase != "combat" || session.BossHP < 150 || session.Difficulty != "easy" || session.QuestionID == "" {
			t.Fatal("event acceptance did not snapshot real challenge", event.ID)
		}
		if session.Question.Topic != TopicAdd && session.Question.Topic != TopicSub {
			t.Fatal("event silently increased mathdifficulty")
		}
		normal := Session{Room: 2, Dungeon: "crypt", Round: 1}
		if incomingHit(&session, 1) <= incomingHit(&normal, 1) {
			t.Fatal("rare event damage is ordinary", event.ID)
		}
		if err := s.enterDungeonEvent(&session, now); err == nil {
			t.Fatal("accepted event twice")
		}
	}
}
func TestFalseGatePhasesAndHiddenShrineMechanics(t *testing.T) {
	session := Session{Room: 2, Rooms: 2, Event: plannedEvent(0, 1), BossHP: 300, BossMaxHP: 300, Combo: 1, Charge: 3, AbilityReady: true}
	session.Event.Accepted = true
	setEventIntent(&session)
	if session.Event.Phase != 1 || eventHit(&session, 30) >= 30 {
		t.Fatal("phase one lacks armor")
	}
	session.Combo = 3
	if eventHit(&session, 30) != 30 {
		t.Fatal("combo does not break event armor")
	}
	session.BossHP = 150
	setEventIntent(&session)
	if session.Event.Phase != 2 {
		t.Fatal("missing second phase")
	}
	session.BossHP = 90
	setEventIntent(&session)
	eventMistake(&session)
	if session.Event.Phase != 3 || session.Charge != 0 || session.AbilityReady {
		t.Fatal("final phase does not drain charge")
	}
	session.Event = plannedEvent(20, 1)
	session.Event.Accepted = true
	session.BossHP = 80
	eventMistake(&session)
	if session.BossHP <= 80 {
		t.Fatal("hidden shrine mistake does not heal guardian")
	}
}
func TestEventFailureCannotPayVictorySeals(t *testing.T) {
	s := &Service{rnd: seeded(2)}
	for _, roll := range []int{0, 8, 20} {
		data := testRow()
		data.Progress.Materials["bone"] = 4
		session := Session{Kind: SessionGate, Dungeon: "crypt", Room: 2, Rooms: 2, HeroHP: 0, BossHP: 40, Event: plannedEvent(roll, 1)}
		session.Event.Accepted = true
		data.Session = &session
		result := s.finishRun(&data, &session, false, time.Now())
		if result.Won || result.Relic != nil || data.Progress.Materials["seal"] != 0 || data.Progress.Materials["bone"] != 4 || data.Progress.Clears != 0 {
			t.Fatal("event failure paid or deleted banked materials")
		}
		session.BossHP = 0
		session.Room = 1
		session.Rooms = 3
		data.Session = &session
		result = s.finishRun(&data, &session, true, time.Now())
		if result.Won || data.Progress.Materials["seal"] != 0 {
			t.Fatal("intermediate room could pay eventclear")
		}
	}
}
