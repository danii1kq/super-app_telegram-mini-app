package farm

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"
)

func gameRequest(i int) string { return fmt.Sprintf("game-request-%016d", i) }
func gameRoll(value int) func(int) (int, error) {
	return func(n int) (int, error) { return value % n, nil }
}

func TestCrashCashoutAndHiddenOutcome(t *testing.T) {
	now := time.Now()
	s := initialState(now)
	s.Coins = 10000
	if _, _, _, err := apply(&s, Action{Kind: "crash_start", Stake: 1000, RequestID: gameRequest(1)}, now, gameRoll(200000)); err != nil {
		t.Fatal(err)
	}
	if s.Coins != 9000 || s.Crash.Status != "running" {
		t.Fatal("stake not escrowed")
	}
	public := publicGameState(s, now)
	data, _ := json.Marshal(public)
	if public.Crash.BustBP != 0 || strings.Contains(string(data), "bust_bp") || strings.Contains(string(data), "game_requests") || s.Crash.BustBP == 0 {
		t.Fatal("unfinished crash secret exposed or erased")
	}
	id := s.Crash.ID
	if _, _, _, err := apply(&s, Action{Kind: "crash_cashout", Item: id, RequestID: gameRequest(2)}, now.Add(10*time.Second), fixed); err != nil {
		t.Fatal(err)
	}
	want := 9000 + 1000*crashMultiplier(now.Add(10*time.Second), now)/100
	if s.Coins != want || s.Crash.Status != "cashed" || len(s.GameHistory) != 1 || s.Sold != 0 {
		t.Fatal("cashout changed the wrong ledger")
	}
	if _, _, _, err := apply(&s, Action{Kind: "crash_cashout", Item: id, RequestID: gameRequest(3)}, now.Add(11*time.Second), fixed); err == nil || s.Coins != want {
		t.Fatal("double cashout")
	}
}

func TestCrashInstantLossAutoCashoutAndRequestReplay(t *testing.T) {
	now := time.Now()
	s := initialState(now)
	s.Coins = 10000
	start := Action{Kind: "crash_start", Stake: 1000, RequestID: gameRequest(1)}
	if _, _, _, err := apply(&s, start, now, gameRoll(999999)); err != nil {
		t.Fatal(err)
	}
	if s.Crash.Status != "busted" || s.Coins != 9000 || len(s.GameHistory) != 1 {
		t.Fatal("instant loss did not settle")
	}
	if _, _, _, err := apply(&s, start, now, fixed); err == nil || s.Coins != 9000 {
		t.Fatal("replayed start charged twice")
	}
	start.RequestID = gameRequest(2)
	if _, _, _, err := apply(&s, start, now, gameRoll(0)); err != nil {
		t.Fatal(err)
	}
	if !refreshCrash(&s, now.Add(30*time.Second)) || s.Coins != 18000 || s.Crash.Payout != 10000 {
		t.Fatal("10x cap not paid")
	}
	if refreshCrash(&s, now.Add(time.Hour)) || s.Coins != 18000 {
		t.Fatal("auto payout repeated")
	}
}

func TestCrashDistributionMatchesPublishedChances(t *testing.T) {
	for bp, expected := range map[int]int{100: 550000, 200: 275000, 300: 183333, 500: 110000, 1000: 55000} {
		wins := 0
		for r := 0; r < 1000000; r++ {
			if crashThreshold(r) >= bp {
				wins++
			}
		}
		if wins != expected {
			t.Fatalf("chance mismatch at %d: %d", bp, wins)
		}
	}
}

func TestUpgradeQuoteConsumptionAndBoundaries(t *testing.T) {
	for _, win := range []bool{false, true} {
		s := initialState(time.Now())
		s.Coins = 10000
		normalizeState(&s)
		s.Upgrades["market"] = 5
		s.Barn = []Harvest{{ID: "source", Seed: "carrot", Quality: "normal", Mutation: "none", Quantity: 3, Value: 96}}
		a := Action{Kind: "plant_upgrade", Item: "source", Seed: "lotus", Upgrade: "crystal", Stake: 1000, RequestID: gameRequest(1)}
		q, err := plantUpgradeQuote(s, a)
		if err != nil {
			t.Fatal(err)
		}
		if q.SourceValue != 40 || q.TargetValue != 18600 || q.ChanceBP != 335 {
			t.Fatalf("quote mismatch %+v", q)
		}
		roll := q.ChanceBP
		if win {
			roll--
		}
		if _, _, _, err = apply(&s, a, time.Now(), gameRoll(roll)); err != nil {
			t.Fatal(err)
		}
		if s.Coins != 9000 || s.Barn[0].Quantity != 2 || s.Barn[0].Value != 64 || s.Sold != 0 {
			t.Fatal("upgrade consumed the wrong amount")
		}
		if win {
			if len(s.Barn) != 2 || s.Barn[1].Mutation != "crystal" || s.Barn[1].Quality != "perfect" || s.Collection["lotus:crystal"].Quality != "perfect" {
				t.Fatal("upgrade reward missing")
			}
		} else if len(s.Barn) != 1 {
			t.Fatal("losing roll granted plant")
		}
		if _, _, _, err = apply(&s, a, time.Now(), fixed); err == nil || s.Coins != 9000 {
			t.Fatal("upgrade request replayed")
		}
	}
}

func TestRebalancedRulesAndExistingCrash(t *testing.T) {
	rules := gameRules()
	if rules["crash_return_percent"] != 55 || rules["flip_fee_percent"] != 25 || rules["upgrade_return_percent"] != 60 || rules["upgrade_max_chance_bp"] != 4500 {
		t.Fatal("published rules do not match rebalanced settings")
	}
	now := time.Now()
	s := initialState(now)
	normalizeState(&s)
	s.Crash = &CrashRound{ID: "old", Stake: 1000, StartedAt: now.Add(-10 * time.Second), Status: "running", BustBP: 480}
	if refreshCrash(&s, now) || publicGameState(s, now).Crash.ReturnPercent != 96 || s.Crash.BustBP != 480 {
		t.Fatal("old crash threshold or published return changed")
	}
	s.Barn = []Harvest{{ID: "source", Seed: "carrot", Quantity: 1, Value: 32}}
	q, err := plantUpgradeQuote(s, Action{Item: "source", Seed: "lotus", Upgrade: "crystal", Stake: 14000})
	if err != nil || q.ChanceBP != 4500 {
		t.Fatal("upgrade maximum is not 45%")
	}
	if coinflipPayout(1000, 25) != 1500 || coinflipPayout(1000, 10) != 1800 || coinflipPayout(101, 25) != 151 {
		t.Fatal("current or legacy coinflip payout changed")
	}
}

func TestInvalidGamesDoNotSpend(t *testing.T) {
	for _, a := range []Action{
		{Kind: "crash_start", Stake: -1}, {Kind: "crash_start", Stake: 100001}, {Kind: "crash_start", Stake: 1000},
		{Kind: "crash_cashout", Item: "invented"}, {Kind: "plant_upgrade", Item: "missing", Seed: "lotus", Upgrade: "crystal", Stake: 100},
	} {
		s := initialState(time.Now())
		a.RequestID = gameRequest(1)
		if _, _, _, err := apply(&s, a, time.Now(), fixed); err == nil || s.Coins != 250 {
			t.Fatalf("invalid action changed coins: %+v", a)
		}
	}
	s := initialState(time.Now())
	s.Barn = []Harvest{{ID: "gold", Seed: "lotus", Quantity: 1, Value: 100000}}
	if _, err := plantUpgradeQuote(s, Action{Item: "gold", Seed: "carrot", Upgrade: "dew", Stake: 100}); err == nil {
		t.Fatal("cheap target accepted")
	}
}
