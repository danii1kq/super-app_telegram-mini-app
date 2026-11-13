package aura

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
)

func TestAuraHTTPFavoritesOwnedLimitAndPersistence(t *testing.T) {
	pool, request := auraHTTPFixture(t)
	readRaidView(t, request("GET", "/api/v1/aura", nil, 1))
	if _, err := pool.Exec(context.Background(), `INSERT INTO aura_relics(user_id,relic_key,rank)VALUES(1,'rusty_blade','E'),(1,'cloth_robe','E'),(1,'copper_ring','E'),(1,'hunters_bow','D')`); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"rusty_blade", "cloth_robe", "copper_ring"} {
		if got := request("POST", "/api/v1/aura/relics/favorite", map[string]string{"key": key}, 1); got.Code != 200 {
			t.Fatal(got.Body.String())
		}
	}
	if request("POST", "/api/v1/aura/relics/favorite", map[string]string{"key": "hunters_bow"}, 1).Code != 400 {
		t.Fatal("fourth display slot accepted")
	}
	if request("POST", "/api/v1/aura/relics/favorite", map[string]string{"key": "world_ender"}, 1).Code != 400 {
		t.Fatal("unowned display item accepted")
	}
	if request("POST", "/api/v1/aura/relics/favorite", map[string]string{"key": "rusty_blade"}, 2).Code != 400 {
		t.Fatal("another user's relic could be displayed")
	}
	readRaidView(t, request("POST", "/api/v1/aura/relics/favorite", map[string]string{"key": "rusty_blade"}, 1))
	readRaidView(t, request("POST", "/api/v1/aura/relics/favorite", map[string]string{"key": "hunters_bow"}, 1))
	var view struct {
		Showcase []struct {
			Key      string `json:"key"`
			Favorite bool   `json:"favorite"`
		} `json:"showcase"`
	}
	if err := json.Unmarshal(request("GET", "/api/v1/aura", nil, 1).Body.Bytes(), &view); err != nil {
		t.Fatal(err)
	}
	if len(view.Showcase) != 3 {
		t.Fatal("showcase not persisted", view)
	}
	for _, item := range view.Showcase {
		if !item.Favorite || item.Key == "rusty_blade" {
			t.Fatal("showcase did not retain exact choice")
		}
	}
}

func TestAuraHTTPFirstBossRewardsCommitOnce(t *testing.T) {
	pool, request := auraHTTPFixture(t)
	ctx := context.Background()
	readRaidView(t, request("POST", "/api/v1/aura/start", map[string]string{"kind": "gate", "dungeon": "crypt"}, 1))
	if _, err := pool.Exec(ctx, `UPDATE aura_players SET session=jsonb_set(jsonb_set(jsonb_set(session-'pending_event','{room}','3'::jsonb),'{rooms}','3'::jsonb),'{boss_hp}','1'::jsonb) WHERE user_id=1`); err != nil {
		t.Fatal(err)
	}
	before := readRaidView(t, request("GET", "/api/v1/aura", nil, 1))
	correct := raidAnswer(t, pool, 1)
	var wg sync.WaitGroup
	statuses := make(chan int, 6)
	for i := 0; i < 6; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			statuses <- request("POST", "/api/v1/aura/answer", map[string]any{"choice": correct, "question_id": before.Player.Session.QuestionID, "request_id": fmt.Sprintf("first-trophy-%d", i)}, 1).Code
		}(i)
	}
	wg.Wait()
	close(statuses)
	success := 0
	for status := range statuses {
		if status == 200 {
			success++
		} else if status != 400 {
			t.Fatal("unexpected final answer status", status)
		}
	}
	if success != 1 {
		t.Fatal("boss paid more than once")
	}
	var count, level int
	if err := pool.QueryRow(ctx, `SELECT count(*),coalesce(max(level),0) FROM aura_relics WHERE user_id=1 AND relic_key='trophy_crypt'`).Scan(&count, &level); err != nil {
		t.Fatal(err)
	}
	if count != 1 || level != 1 {
		t.Fatal("trophy duplicated or upgraded by replay", count, level)
	}
	var progress Progress
	var raw []byte
	if err := pool.QueryRow(ctx, `SELECT progress FROM aura_players WHERE user_id=1`).Scan(&raw); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &progress); err != nil {
		t.Fatal(err)
	}
	if !progress.Discoveries["trophy_crypt"] || !progress.Discoveries["vow_unbroken"] {
		t.Fatal("one-time rewards not marked")
	}
	if _, err := pool.Exec(ctx, `UPDATE aura_players SET gate_ready_at=NULL WHERE user_id=1`); err != nil {
		t.Fatal(err)
	}
	readRaidView(t, request("POST", "/api/v1/aura/start", map[string]string{"kind": "gate", "dungeon": "crypt"}, 1))
	if _, err := pool.Exec(ctx, `UPDATE aura_players SET session=jsonb_set(jsonb_set(jsonb_set(session-'pending_event','{room}','3'::jsonb),'{rooms}','3'::jsonb),'{boss_hp}','1'::jsonb) WHERE user_id=1`); err != nil {
		t.Fatal(err)
	}
	next := readRaidView(t, request("GET", "/api/v1/aura", nil, 1))
	readRaidView(t, request("POST", "/api/v1/aura/answer", map[string]any{"choice": raidAnswer(t, pool, 1), "question_id": next.Player.Session.QuestionID, "request_id": "second-trophy-attempt"}, 1))
	if err := pool.QueryRow(ctx, `SELECT level FROM aura_relics WHERE user_id=1 AND relic_key='trophy_crypt'`).Scan(&level); err != nil || level != 1 {
		t.Fatal("repeat victory awarded same trophy twice", level, err)
	}
}
