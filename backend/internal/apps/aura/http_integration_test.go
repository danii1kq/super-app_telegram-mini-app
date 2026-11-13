package aura

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/danii1kq/srunal/internal/core/auth"
	"github.com/danii1kq/srunal/internal/platform/postgres"
	"github.com/danii1kq/srunal/internal/platform/secure"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type raidTestSession struct {
	Question struct {
		Options []int `json:"options"`
	} `json:"question"`
	QuestionID   string `json:"question_id"`
	Phase        string `json:"phase"`
	Round        int    `json:"round"`
	Room         int    `json:"room"`
	HeroHP       int    `json:"hero_hp"`
	BossHP       int    `json:"boss_hp"`
	AbilityReady bool   `json:"ability_ready"`
}

type raidTestView struct {
	Player struct {
		Aura      int64            `json:"aura"`
		Clears    int              `json:"clears"`
		Materials map[string]int   `json:"materials"`
		Session   *raidTestSession `json:"session"`
	} `json:"player"`
	Relics []json.RawMessage `json:"relics"`
}

func auraHTTPFixture(t *testing.T) (*pgxpool.Pool, func(string, string, any, int) *httptest.ResponseRecorder) {
	t.Helper()
	dsn := os.Getenv("SRUNAL_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set SRUNAL_TEST_DATABASE_URL for isolated PostgreSQL tests")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	schema := pgx.Identifier{"aura_test_" + strconv.FormatInt(time.Now().UnixNano(), 10)}.Sanitize()
	if _, err = admin.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		admin.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = admin.Exec(context.Background(), "DROP SCHEMA "+schema+" CASCADE"); admin.Close() })
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if _, err = pool.Exec(ctx, `CREATE TABLE users(id bigint PRIMARY KEY,public_name text NOT NULL,username text,status text NOT NULL DEFAULT 'active');
	CREATE TABLE sessions(token_hash text PRIMARY KEY,user_id bigint,revoked_at timestamptz,expires_at timestamptz);
	INSERT INTO users(id,public_name)VALUES(1,'Hunter'),(2,'Other'),(3,'Third');`); err != nil {
		t.Fatal(err)
	}
	for _, file := range []string{"000021_aura.up.sql", "000022_aura_dungeons.up.sql"} {
		data, err := os.ReadFile("../../../../db/migrations/" + file)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = pool.Exec(ctx, string(data)); err != nil {
			t.Fatal(err)
		}
	}
	for id := 1; id <= 3; id++ {
		if _, err = pool.Exec(ctx, `INSERT INTO sessions(token_hash,user_id,expires_at)VALUES($1,$2,now()+interval '1 hour')`, secure.TokenHash(fmt.Sprintf("aura-test-%d", id)), id); err != nil {
			t.Fatal(err)
		}
	}
	db := &postgres.Pool{Pool: pool}
	mux := http.NewServeMux()
	NewHandler(NewService(db), auth.NewService(db, nil, nil, time.Hour)).Register(mux)
	request := func(method, path string, body any, user int) *httptest.ResponseRecorder {
		data, _ := json.Marshal(body)
		r := httptest.NewRequest(method, path, strings.NewReader(string(data)))
		if user > 0 {
			r.Header.Set("Authorization", fmt.Sprintf("Bearer aura-test-%d", user))
		}
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		return w
	}
	return pool, request
}

func readRaidView(t *testing.T, w *httptest.ResponseRecorder) raidTestView {
	t.Helper()
	if w.Code != 200 {
		t.Fatalf("HTTP %d: %s", w.Code, w.Body.String())
	}
	var view raidTestView
	if err := json.Unmarshal(w.Body.Bytes(), &view); err != nil {
		t.Fatal(err)
	}
	return view
}

func raidAnswer(t *testing.T, pool *pgxpool.Pool, user int) int {
	t.Helper()
	var correct int
	if err := pool.QueryRow(context.Background(), `SELECT (session->'question'->>'answer')::int FROM aura_players WHERE user_id=$1`, user).Scan(&correct); err != nil {
		t.Fatal(err)
	}
	return correct
}

func TestAuraHTTPAnswerRetriesAndIsolation(t *testing.T) {
	pool, request := auraHTTPFixture(t)
	if request("GET", "/api/v1/aura", nil, 0).Code != 401 {
		t.Fatal("unauthenticated state allowed")
	}
	readRaidView(t, request("GET", "/api/v1/aura", nil, 1))
	if request("POST", "/api/v1/aura/start", map[string]any{"kind": "gate", "dungeon": "void"}, 1).Code != 400 {
		t.Fatal("locked dungeon accepted")
	}
	readRaidView(t, request("POST", "/api/v1/aura/start", map[string]any{"kind": "gate", "dungeon": "crypt"}, 1))
	if _, err := pool.Exec(context.Background(), `UPDATE aura_players SET session=session-'pending_event' WHERE user_id=1`); err != nil {
		t.Fatal(err)
	}
	before := readRaidView(t, request("GET", "/api/v1/aura", nil, 1))
	if before.Player.Session == nil || before.Player.Session.QuestionID == "" {
		t.Fatal("missing server question identity")
	}
	qid := before.Player.Session.QuestionID
	correct := raidAnswer(t, pool, 1)
	var wg sync.WaitGroup
	statuses := make(chan int, 8)
	for n := 0; n < 8; n++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			statuses <- request("POST", "/api/v1/aura/answer", map[string]any{"choice": correct, "question_id": qid, "request_id": fmt.Sprintf("parallel-answer-%d", n)}, 1).Code
		}(n)
	}
	wg.Wait()
	close(statuses)
	success := 0
	for status := range statuses {
		if status == 200 {
			success++
		} else if status != 400 && status != 409 {
			t.Fatalf("unexpected concurrent answer status %d", status)
		}
	}
	after := readRaidView(t, request("GET", "/api/v1/aura", nil, 1))
	if success != 1 || after.Player.Session == nil || after.Player.Session.Round != before.Player.Session.Round+1 {
		t.Fatal("concurrent requests advanced more than one question")
	}
	if other := readRaidView(t, request("GET", "/api/v1/aura", nil, 2)); other.Player.Session != nil || other.Player.Clears != 0 {
		t.Fatal("run leaked into another account")
	}
	saved := after.Player.Session.QuestionID
	request("POST", "/api/v1/aura/answer", map[string]any{"choice": correct, "question_id": qid, "request_id": "parallel-answer-0"}, 1)
	if afterRetry := readRaidView(t, request("GET", "/api/v1/aura", nil, 1)); afterRetry.Player.Session.QuestionID != saved {
		t.Fatal("replayed answer advanced a new question")
	}
}

func TestAuraHTTPDungeonVictoryAndDefeat(t *testing.T) {
	pool, request := auraHTTPFixture(t)
	readRaidView(t, request("GET", "/api/v1/aura", nil, 1))
	// Only isolated fixtures are strengthened, to reach route and boss transitions reliably.
	if _, err := pool.Exec(context.Background(), `UPDATE aura_players SET stat_str=15,stat_vit=10 WHERE user_id=1`); err != nil {
		t.Fatal(err)
	}
	readRaidView(t, request("POST", "/api/v1/aura/start", map[string]any{"kind": "gate", "dungeon": "crypt"}, 1))
	for n := 0; n < 50; n++ {
		if n == 0 {
			if _, err := pool.Exec(context.Background(), `UPDATE aura_players SET session=session-'pending_event' WHERE user_id=1`); err != nil {
				t.Fatal(err)
			}
		}
		view := readRaidView(t, request("GET", "/api/v1/aura", nil, 1))
		session := view.Player.Session
		if session == nil {
			break
		}
		if session.Phase == "choice" {
			readRaidView(t, request("POST", "/api/v1/aura/route", map[string]any{"route": "normal", "request_id": fmt.Sprintf("route-%d", n)}, 1))
		} else {
			readRaidView(t, request("POST", "/api/v1/aura/answer", map[string]any{"choice": raidAnswer(t, pool, 1), "question_id": session.QuestionID, "request_id": fmt.Sprintf("win-%d", n)}, 1))
		}
	}
	won := readRaidView(t, request("GET", "/api/v1/aura", nil, 1))
	if won.Player.Session != nil || won.Player.Clears != 1 || len(won.Relics) < 2 || len(won.Relics) > 3 {
		t.Fatalf("boss victory failed to grant one clear, trophy and clean-run reward: %+v", won)
	}
	var weeklyClears int
	if err := pool.QueryRow(context.Background(), `SELECT clears FROM aura_weekly_scores WHERE user_id=1`).Scan(&weeklyClears); err != nil || weeklyClears != 1 {
		t.Fatal("victory missing from weekly ranking", weeklyClears, err)
	}
	claimed := readRaidView(t, request("POST", "/api/v1/aura/quests/claim", map[string]any{"id": "first_clear"}, 1))
	if claimed.Player.Aura != won.Player.Aura+250 {
		t.Fatal("first clear quest paid wrong amount")
	}
	if request("POST", "/api/v1/aura/quests/claim", map[string]any{"id": "first_clear"}, 1).Code != 400 {
		t.Fatal("quest paid twice")
	}
	readRaidView(t, request("POST", "/api/v1/aura/start", map[string]any{"kind": "gate", "dungeon": "crypt"}, 2))
	for n := 0; n < 40; n++ {
		view := readRaidView(t, request("GET", "/api/v1/aura", nil, 2))
		if view.Player.Session == nil {
			break
		}
		correct := raidAnswer(t, pool, 2)
		wrong := correct
		for _, option := range view.Player.Session.Question.Options {
			if option != correct {
				wrong = option
				break
			}
		}
		if wrong == correct {
			t.Fatal("question has no wrong option")
		}
		readRaidView(t, request("POST", "/api/v1/aura/answer", map[string]any{"choice": wrong, "question_id": view.Player.Session.QuestionID, "request_id": fmt.Sprintf("lose-%d", n)}, 2))
	}
	lost := readRaidView(t, request("GET", "/api/v1/aura", nil, 2))
	if lost.Player.Session != nil || lost.Player.Clears != 0 || len(lost.Relics) != 0 {
		t.Fatal("failed dungeon granted a boss reward")
	}
}

func TestAuraHTTPAbilityAndTimeout(t *testing.T) {
	pool, request := auraHTTPFixture(t)
	readRaidView(t, request("POST", "/api/v1/aura/class", map[string]any{"class": "strategist"}, 1))
	readRaidView(t, request("POST", "/api/v1/aura/start", map[string]any{"kind": "gate", "dungeon": "crypt"}, 1))
	if _, err := pool.Exec(context.Background(), `UPDATE aura_players SET session=session-'pending_event' WHERE user_id=1`); err != nil {
		t.Fatal(err)
	}
	var current raidTestView
	for n := 0; n < 12; n++ {
		current = readRaidView(t, request("GET", "/api/v1/aura", nil, 1))
		s := current.Player.Session
		if s == nil {
			t.Fatal("run ended before charged ability")
		}
		if s.Phase == "choice" {
			readRaidView(t, request("POST", "/api/v1/aura/route", map[string]any{"route": "normal", "request_id": fmt.Sprintf("charge-route-%d", n)}, 1))
			continue
		}
		if s.AbilityReady {
			break
		}
		readRaidView(t, request("POST", "/api/v1/aura/answer", map[string]any{"choice": raidAnswer(t, pool, 1), "question_id": s.QuestionID, "request_id": fmt.Sprintf("charge-%d", n)}, 1))
	}
	if current.Player.Session == nil || !current.Player.Session.AbilityReady {
		t.Fatal("three correct answers did not charge ability")
	}
	activated := readRaidView(t, request("POST", "/api/v1/aura/ability", map[string]any{"request_id": "ability-once"}, 1))
	if activated.Player.Session == nil || activated.Player.Session.AbilityReady || len(activated.Player.Session.Question.Options) != 2 {
		t.Fatal("strategist ability did not remove two options and spend charge")
	}
	if request("POST", "/api/v1/aura/ability", map[string]any{"request_id": "ability-once"}, 1).Code != 400 {
		t.Fatal("ability request replay accepted")
	}
	if _, err := pool.Exec(context.Background(), `UPDATE aura_players SET session=jsonb_set(session,'{question_deadline}',to_jsonb(now()-interval '1 second')) WHERE user_id=1`); err != nil {
		t.Fatal(err)
	}
	q := activated.Player.Session
	timedOut := readRaidView(t, request("POST", "/api/v1/aura/answer", map[string]any{"choice": raidAnswer(t, pool, 1), "question_id": q.QuestionID, "request_id": "late-answer"}, 1))
	if timedOut.Player.Session == nil || timedOut.Player.Session.HeroHP >= q.HeroHP || timedOut.Player.Session.BossHP != q.BossHP {
		t.Fatal("late correct answer did not apply and persist enemy damage")
	}
}

func TestAuraHTTPCraftingIsAtomic(t *testing.T) {
	pool, request := auraHTTPFixture(t)
	readRaidView(t, request("GET", "/api/v1/aura", nil, 1))
	if _, err := pool.Exec(context.Background(), `UPDATE aura_players SET aura=5000,progress=jsonb_set(progress,'{materials}','{"bone":100,"ember":100,"void":100}'::jsonb) WHERE user_id=1`); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	statuses := make(chan int, 6)
	for n := 0; n < 6; n++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			statuses <- request("POST", "/api/v1/aura/relics/craft", map[string]any{"key": "echo_blade"}, 1).Code
		}()
	}
	wg.Wait()
	close(statuses)
	success := 0
	for status := range statuses {
		if status == 200 {
			success++
		} else if status != 400 {
			t.Fatalf("craft status %d", status)
		}
	}
	view := readRaidView(t, request("GET", "/api/v1/aura", nil, 1))
	if success != 1 || len(view.Relics) != 1 {
		t.Fatal("concurrent craft duplicated item or material spending", success, view.Player.Materials)
	}
	var response struct {
		Catalog []struct {
			Key       string         `json:"key"`
			CraftAura int64          `json:"craft_aura"`
			CraftCost map[string]int `json:"craft_cost"`
		} `json:"catalog"`
	}
	if err := json.Unmarshal(request("GET", "/api/v1/aura", nil, 1).Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	for _, item := range response.Catalog {
		if item.Key == "echo_blade" {
			if view.Player.Aura != 5000-item.CraftAura || view.Player.Materials["bone"] != 100-item.CraftCost["bone"] {
				t.Fatal("craft spending differs from quoted price")
			}
		}
	}
	if request("POST", "/api/v1/aura/relics/craft", map[string]any{"key": "world_ender"}, 1).Code != 400 {
		t.Fatal("exclusive relic could be crafted for free")
	}
}

func TestAuraHTTPLegacyInventoryAndPaidUpgrade(t *testing.T) {
	pool, request := auraHTTPFixture(t)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `INSERT INTO aura_players(user_id,aura,total_aura,level,xp,stat_str) VALUES(3,10000,12000,2,90,3);
	INSERT INTO aura_relics(user_id,relic_key,rank,level,slot) VALUES(3,'rusty_blade','E',2,'weapon');`); err != nil {
		t.Fatal(err)
	}
	type relicView struct {
		Key   string  `json:"key"`
		Level int     `json:"level"`
		Value float64 `json:"value"`
		Cost  int64   `json:"upgrade_cost"`
		Worn  bool    `json:"worn"`
	}
	readRelic := func() relicView {
		w := request("GET", "/api/v1/aura", nil, 3)
		view := readRaidView(t, w)
		if len(view.Relics) != 1 {
			t.Fatal("legacy inventory was lost")
		}
		var relic relicView
		if err := json.Unmarshal(view.Relics[0], &relic); err != nil {
			t.Fatal(err)
		}
		return relic
	}
	old := readRelic()
	if old.Level != 2 || !old.Worn || old.Cost <= 0 {
		t.Fatal("legacy item level, slot or price lost", old)
	}
	w := request("POST", "/api/v1/aura/relics/upgrade", map[string]any{"key": "rusty_blade"}, 3)
	if w.Code != 200 {
		t.Fatalf("upgrade failed: %s", w.Body.String())
	}
	newRelic := readRelic()
	var coins int64
	if err := pool.QueryRow(ctx, `SELECT aura FROM aura_players WHERE user_id=3`).Scan(&coins); err != nil {
		t.Fatal(err)
	}
	if newRelic.Level != 3 || newRelic.Value <= old.Value || coins != 10000-old.Cost || !newRelic.Worn {
		t.Fatal("paid upgrade did not spend exact quote and increase equipped effect", old, newRelic, coins)
	}
	if _, err := pool.Exec(ctx, `UPDATE aura_players SET aura=0 WHERE user_id=3`); err != nil {
		t.Fatal(err)
	}
	if request("POST", "/api/v1/aura/relics/upgrade", map[string]any{"key": "rusty_blade"}, 3).Code != 400 {
		t.Fatal("unaffordable upgrade accepted")
	}
	if readRelic().Level != 3 {
		t.Fatal("failed upgrade changed level")
	}
	if request("POST", "/api/v1/aura/relics/equip", map[string]any{"key": "rusty_blade", "slot": "armor"}, 3).Code != 400 {
		t.Fatal("weapon equipped as armor")
	}
}
