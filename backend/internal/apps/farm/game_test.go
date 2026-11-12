package farm

import (
	"testing"
	"time"
)

func fixed(n int) (int, error) { return 99 % n, nil }

func TestPlantHarvestSellCycle(t *testing.T) {
	now := time.Date(2026, 11, 12, 12, 0, 0, 0, time.UTC)
	s := initialState(now)
	s.Plots = make([]*Plot, 4)
	_, _, _, err := apply(&s, Action{Kind: "plant", Slot: 0, Seed: "carrot", Care: "water"}, now, fixed)
	if err != nil {
		t.Fatal(err)
	}
	if s.Seeds["carrot"] != 3 || s.Coins != 250 {
		t.Fatal("plant charged wrong amount")
	}
	if _, _, _, err = apply(&s, Action{Kind: "harvest"}, now.Add(GrowTime-time.Second), fixed); err == nil {
		t.Fatal("early harvest accepted")
	}
	_, _, _, err = apply(&s, Action{Kind: "harvest"}, now.Add(GrowTime), fixed)
	if err != nil {
		t.Fatal(err)
	}
	if s.Plots[0] != nil || len(s.Barn) != 1 || len(s.Collection) != 1 {
		t.Fatal("harvest not recorded")
	}
	_, _, revenue, err := apply(&s, Action{Kind: "sell"}, now, fixed)
	if err != nil || revenue != 32 || s.Coins != 282 || len(s.Barn) != 0 {
		t.Fatalf("wrong sale: %d %v", revenue, err)
	}
	if _, _, _, err = apply(&s, Action{Kind: "sell"}, now, fixed); err == nil {
		t.Fatal("double sale accepted")
	}
}

func TestInvalidActionsDoNotSpend(t *testing.T) {
	for _, action := range []Action{
		{Kind: "plant", Slot: -1, Seed: "carrot", Care: "water"},
		{Kind: "plant", Slot: 4, Seed: "carrot", Care: "water"},
		{Kind: "plant", Slot: 0, Seed: "carrot", Care: "water"},
		{Kind: "plant", Slot: 2, Seed: "unknown", Care: "water"},
		{Kind: "plant", Slot: 2, Seed: "carrot", Care: "unknown"},
		{Kind: "buy", Seed: "dragonfruit"},
		{Kind: "pack", Pack: "exotic"},
		{Kind: "upgrade", Upgrade: "unknown"},
		{Kind: "reset"},
	} {
		s := initialState(time.Now())
		s.Coins = 10
		if _, _, _, err := apply(&s, action, time.Now(), fixed); err == nil {
			t.Fatalf("accepted %+v", action)
		}
		if s.Coins != 10 {
			t.Fatalf("charged for invalid action %+v", action)
		}
	}
}

func TestOutcomesAndSnapshot(t *testing.T) {
	plot := Plot{Seed: "carrot", Care: "water"}
	if got := outcome(plot, 9, 99); got.Quality != "perfect" || got.Value != 96 {
		t.Fatalf("wrong perfect outcome: %+v", got)
	}
	if got := outcome(plot, 10, 99); got.Quality != "choice" {
		t.Fatal("quality boundary")
	}
	if got := outcome(plot, 40, 99); got.Quality != "normal" {
		t.Fatal("normal boundary")
	}
	if got := outcome(plot, 99, 0); got.Mutation != "crystal" || got.Value != 256 {
		t.Fatal("crystal roll")
	}
	if got := outcome(Plot{Seed: "carrot", Lamp: 1}, 99, 1); got.Mutation != "gold" {
		t.Fatal("gold roll")
	}
	if got := outcome(Plot{Seed: "carrot", Soil: 2, Care: "soil"}, 99, 99); got.Quantity != 4 {
		t.Fatal("fertilizer yield")
	}
	s := initialState(time.Now())
	_, _, _, err := apply(&s, Action{Kind: "plant", Slot: 2, Seed: "carrot", Care: "water"}, time.Now(), fixed)
	if err != nil {
		t.Fatal(err)
	}
	_, _, _, err = apply(&s, Action{Kind: "upgrade", Upgrade: "water"}, time.Now(), fixed)
	if err != nil || s.Plots[2].Water != 0 || s.Upgrades["water"] != 1 {
		t.Fatal("upgrade changed planted crop")
	}
}

func TestPackGuaranteeAndUpgradeCap(t *testing.T) {
	s := initialState(time.Now())
	s.Coins = 10000
	s.Pity["garden"] = 8
	zero := func(int) (int, error) { return 0, nil }
	_, drops, _, err := apply(&s, Action{Kind: "pack", Pack: "garden"}, time.Now(), zero)
	if err != nil || len(drops) != 3 || drops[1] != "blueberry" || s.Pity["garden"] != 1 {
		t.Fatalf("guarantee failed: %v %+v %v", drops, s.Pity, err)
	}
	if s.Pity["exotic"] != 0 {
		t.Fatal("guarantee leaked across packs")
	}
	s.Upgrades["water"] = 5
	coins := s.Coins
	if _, _, _, err = apply(&s, Action{Kind: "upgrade", Upgrade: "water"}, time.Now(), fixed); err == nil || s.Coins != coins {
		t.Fatal("upgrade cap ignored")
	}
	for _, pack := range Packs {
		sum := 0
		for _, w := range pack.Weights {
			sum += w
		}
		if sum != 100 || len(pack.Weights) != len(Crops) {
			t.Fatal("invalid odds")
		}
	}
}

func TestWeekStart(t *testing.T) {
	for day := 9; day <= 15; day++ {
		got := weekStart(time.Date(2026, 11, day, 23, 59, 0, 0, time.UTC))
		if got.Day() != 9 || got.Hour() != 0 {
			t.Fatalf("wrong week for day %d: %v", day, got)
		}
	}
}
