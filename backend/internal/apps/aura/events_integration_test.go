package aura

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

type eventHTTPView struct {
	Player struct {
		Session   *Session       `json:"session"`
		Materials map[string]int `json:"materials"`
		Clears    int            `json:"clears"`
	} `json:"player"`
	Won      bool `json:"won"`
	Finished bool `json:"finished"`
}

func TestAuraHTTPEventConsentWinAndNoReplay(t *testing.T) {
	pool, request := auraHTTPFixture(t)
	ctx := context.Background()
	for index, roll := range []int{0, 8, 20} {
		user := index + 1
		if got := request("POST", "/api/v1/aura/start", map[string]any{"kind": "gate", "dungeon": "crypt"}, user); got.Code != 200 {
			t.Fatal(got.Body.String())
		}
		event := plannedEvent(roll, 1)
		encoded, _ := json.Marshal(event)
		if _, err := pool.Exec(ctx, `UPDATE aura_players SET session=jsonb_set(jsonb_set(session,'{pending_event}',$2::jsonb),'{boss_hp}','1'::jsonb) WHERE user_id=$1`, user, encoded); err != nil {
			t.Fatal(err)
		}
		var saved string
		if err := pool.QueryRow(ctx, `SELECT session->>'pending_event' FROM aura_players WHERE user_id=$1`, user).Scan(&saved); err != nil {
			t.Fatal(err)
		}
		for i := 0; i < 3; i++ {
			state := request("GET", "/api/v1/aura", nil, user)
			if state.Code != 200 || strings.Contains(state.Body.String(), "pending_event") || strings.Contains(state.Body.String(), event.Name) {
				t.Fatal("future event leaked before reveal", state.Body.String())
			}
		}
		var after string
		_ = pool.QueryRow(ctx, `SELECT session->>'pending_event' FROM aura_players WHERE user_id=$1`, user).Scan(&after)
		if saved != after {
			t.Fatal("GET rerolled pending event")
		}
		state := request("GET", "/api/v1/aura", nil, user)
		var before eventHTTPView
		_ = json.Unmarshal(state.Body.Bytes(), &before)
		reveal := request("POST", "/api/v1/aura/answer", map[string]any{"choice": raidAnswer(t, pool, user), "question_id": before.Player.Session.QuestionID, "request_id": fmt.Sprintf("event-reveal-%d", user)}, user)
		var view eventHTTPView
		_ = json.Unmarshal(reveal.Body.Bytes(), &view)
		if reveal.Code != 200 || view.Player.Session == nil || view.Player.Session.Phase != "event" || view.Player.Session.Event == nil || view.Player.Session.Event.Accepted {
			t.Fatal("missing consent phase", reveal.Body.String())
		}
		if request("POST", "/api/v1/aura/route", map[string]string{"route": "normal", "request_id": "bypass-consent"}, user).Code != 400 {
			t.Fatal("bypassed event consent")
		}
		enterID := fmt.Sprintf("enter-%d", user)
		entered := request("POST", "/api/v1/aura/route", map[string]string{"route": "event_enter", "request_id": enterID}, user)
		if entered.Code != 200 {
			t.Fatal(entered.Body.String())
		}
		_ = json.Unmarshal(entered.Body.Bytes(), &view)
		if view.Player.Session == nil || !view.Player.Session.Event.Accepted || view.Player.Session.BossHP < 150 {
			t.Fatal("accepted event lacks stronger enemy")
		}
		if request("POST", "/api/v1/aura/route", map[string]string{"route": "event_enter", "request_id": enterID}, user).Code != 400 {
			t.Fatal("replayed event entry")
		}
		for turn := 0; turn < 80 && view.Player.Session != nil; turn++ {
			var path string
			var payload any
			if view.Player.Session.Phase == "choice" {
				path = "/api/v1/aura/route"
				payload = map[string]string{"route": "normal", "request_id": fmt.Sprintf("continue-%d-%d", user, turn)}
			} else {
				path = "/api/v1/aura/answer"
				payload = map[string]any{"choice": raidAnswer(t, pool, user), "question_id": view.Player.Session.QuestionID, "request_id": fmt.Sprintf("event-hit-%d-%d", user, turn)}
			}
			step := request("POST", path, payload, user)
			if step.Code != 200 {
				t.Fatal(step.Body.String())
			}
			_ = json.Unmarshal(step.Body.Bytes(), &view)
		}
		if view.Player.Session != nil || !view.Won || view.Player.Materials["seal"] != eventSeals(&Session{Event: &DungeonEvent{ID: event.ID, Accepted: true}}) || view.Player.Clears != 1 {
			t.Fatal("event victory did not settle once", event.ID, view.Player.Materials)
		}
	}
}
func TestAuraHTTPEventDeclineKeepsBankedMaterials(t *testing.T) {
	pool, request := auraHTTPFixture(t)
	ctx := context.Background()
	_ = request("POST", "/api/v1/aura/start", map[string]string{"kind": "gate"}, 1)
	event := plannedEvent(0, 1)
	encoded, _ := json.Marshal(event)
	if _, err := pool.Exec(ctx, `UPDATE aura_players SET session=jsonb_set(jsonb_set(session,'{event}',$1::jsonb),'{phase}','"event"'::jsonb),progress=jsonb_set(progress,'{materials}','{"bone":9}'::jsonb) WHERE user_id=1`, encoded); err != nil {
		t.Fatal(err)
	}
	response := request("POST", "/api/v1/aura/route", map[string]string{"route": "event_leave", "request_id": "decline-event"}, 1)
	var view eventHTTPView
	_ = json.Unmarshal(response.Body.Bytes(), &view)
	if response.Code != 200 || view.Player.Session != nil || view.Won || view.Player.Materials["bone"] != 9 || view.Player.Materials["seal"] != 0 {
		t.Fatal("decline lost bank or gave bossloot", response.Body.String())
	}
	if request("POST", "/api/v1/aura/route", map[string]string{"route": "event_leave", "request_id": "decline-event"}, 1).Code != 400 {
		t.Fatal("decline replayed")
	}
}
