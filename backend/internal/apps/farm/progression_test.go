package farm

import (
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestProgressionGatesPreserveCoinsAndOwnedSeeds(t *testing.T) {
	now := time.Now()
	for _, action := range []Action{
		{Kind: "buy", Seed: "starfruit"}, {Kind: "pack", Pack: "orchard"},
		{Kind: "start_expedition", Item: "astral"}, {Kind: "upgrade", Upgrade: "soil"},
	} {
		s := initialState(now)
		normalizeState(&s)
		s.Coins = 10000000
		s.Upgrades["soil"] = 5
		before, _ := json.Marshal(s)
		if _, _, _, err := apply(&s, action, now, fixed); err == nil {
			t.Fatalf("accepted locked action: %+v", action)
		}
		after, _ := json.Marshal(s)
		if string(before) != string(after) {
			t.Fatalf("locked action changed save: %+v", action)
		}
	}
	s := initialState(now)
	s.Seeds["starfruit"] = 1
	if _, _, _, err := apply(&s, Action{Kind: "plant", Seed: "starfruit", Slot: 2, Care: "water"}, now, fixed); err != nil {
		t.Fatal("previously owned or lucky pack seeds must remain plantable", err)
	}
	s.XP = 8740 // Level 20, exactly.
	if levelFor(s.XP) != 20 {
		t.Fatal("wrong boundary fixture")
	}
	s.Coins = 800
	if _, _, _, err := apply(&s, Action{Kind: "buy", Seed: "starfruit"}, now, fixed); err != nil || s.Coins != 0 || s.Seeds["starfruit"] != 1 {
		t.Fatal("unlocked seed was not bought at the displayed price", err)
	}
}

func TestNewYieldAndLegacyHarvestSurviveReload(t *testing.T) {
	now := time.Now()
	for _, rollValue := range []int{24, 25} {
		s := initialState(now)
		s.Upgrades["soil"] = 0
		if _, _, _, err := apply(&s, Action{Kind: "plant", Seed: "carrot", Slot: 2, Care: "soil"}, now, func(int) (int, error) { return rollValue, nil }); err != nil {
			t.Fatal(err)
		}
		want := 1
		if rollValue == 24 {
			want = 2
		}
		if s.Plots[2].Yield != want {
			t.Fatal("25 percent care boundary", s.Plots[2].Yield)
		}
		s.Upgrades["soil"] = 20
		data, _ := json.Marshal(s)
		var restored State
		if err := json.Unmarshal(data, &restored); err != nil {
			t.Fatal(err)
		}
		if crop := outcome(*restored.Plots[2], 99, 9999, 99); crop.Quantity != want || crop.Value != 32*want {
			t.Fatal("reload or later upgrade changed a planted promise", crop)
		}
	}
	legacy := Plot{Seed: "starfruit", Soil: 5, Water: 5, Lamp: 5, Genetics: 5, Care: "soil"}
	if got := outcome(legacy, 99, 99, 99); got.Quantity != 7 || got.Value != 9800 {
		t.Fatal("existing planted crops lost their old yield", got)
	}
	s := initialState(now)
	s.Barn = []Harvest{{ID: "old-value", Quantity: 2, Value: 123456}}
	s.Upgrades["market"] = 5
	_, _, revenue, err := apply(&s, Action{Kind: "sell"}, now, fixed)
	if err != nil || revenue != 154320 {
		t.Fatal("existing barn value was recalculated", revenue, err)
	}
}

func TestPlantRandomFailureDoesNotSpend(t *testing.T) {
	s := initialState(time.Now())
	normalizeState(&s)
	before, _ := json.Marshal(s)
	if _, _, _, err := apply(&s, Action{Kind: "plant", Seed: "carrot", Slot: 2, Care: "soil"}, time.Now(), func(int) (int, error) { return 0, errors.New("random source unavailable") }); err == nil {
		t.Fatal("random failure accepted")
	}
	after, _ := json.Marshal(s)
	if string(before) != string(after) {
		t.Fatal("failed planting consumed resources")
	}
}

func TestExtendedUpgradeEconomyHasBoundedEffectsAndLongTermCosts(t *testing.T) {
	total, steps := 0, 0
	for _, upgrade := range Upgrades {
		previous := 0
		for level := 1; level <= upgrade.MaxLevel; level++ {
			cost := upgradeCost(upgrade, level)
			if cost <= previous || upgradeRequiredLevel(level) > MaxLevel {
				t.Fatalf("unreachable or non-increasing upgrade %s level %d", upgrade.ID, level)
			}
			total += cost
			steps++
			previous = cost
		}
	}
	if total < 40000000 || total > 70000000 || steps != 133 {
		t.Fatal("long term progression budget drifted", total, steps)
	}
	if soilPercent(5) != 75 || soilPercent(20) != 150 || marketPercent(5) != 25 || marketPercent(20) != 40 || nurseryPercent(20) != 65 {
		t.Fatal("economy exceeded intended bonuses")
	}
	newPlot := Plot{Yield: 3, Seed: "starfruit", Water: 20, Lamp: 20, Genetics: 10, Care: "lamp"}
	if mutationChanceBP(newPlot) != 4000 || outcome(newPlot, 99, 3999, 99).Mutation != "rainbow" || outcome(newPlot, 99, 4000, 99).Mutation != "none" {
		t.Fatal("new fractional mutation chance boundary")
	}
	if perfectPercent(20)+choicePercent(20)+15 > 100 {
		t.Fatal("quality chances plus water care exceed 100 percent")
	}
	t.Logf("permanent upgrades: %d steps, %d coins", steps, total)
}
