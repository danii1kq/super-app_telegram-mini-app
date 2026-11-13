package farm

import (
	"testing"
	"time"
)

func TestMutationUnlocksAndMultipliers(t *testing.T) {
	for level := 0; level <= 5; level++ {
		seen := map[string]bool{}
		for roll := 0; roll < 100; roll++ {
			id, multiplier := chooseMutation(level, roll)
			seen[id] = true
			crop := outcome(Plot{Seed: "carrot", Genetics: level}, 99, 0, roll)
			if crop.Mutation != id || crop.Value != 32*multiplier {
				t.Fatalf("wrong mutation value: %+v", crop)
			}
		}
		for _, kind := range mutationCatalog(level) {
			if kind.Unlocked != (level >= kind.MinGenetics) || seen[kind.ID] != kind.Unlocked {
				t.Fatalf("level %d unlock mismatch: %+v", level, kind)
			}
		}
	}
	plot := Plot{Seed: "carrot", Lamp: 5, Genetics: 5, Care: "lamp"}
	if mutationChance(plot) != 28 || outcome(plot, 99, 27, 99).Mutation != "rainbow" || outcome(plot, 99, 28, 99).Mutation != "none" {
		t.Fatal("mutation chance boundary or care bonus is wrong")
	}
}

func TestNurseryRecoveryBoundaryAndPlantingSnapshot(t *testing.T) {
	now := time.Now()
	for _, recovery := range []int{49, 50} {
		s := initialState(now)
		s.Plots = make([]*Plot, 4)
		s.Upgrades["genetics"], s.Upgrades["nursery"] = 3, 5
		if _, _, _, err := apply(&s, Action{Kind: "plant", Slot: 0, Seed: "carrot", Care: "water"}, now, fixed); err != nil {
			t.Fatal(err)
		}
		s.Upgrades["genetics"], s.Upgrades["nursery"] = 5, 0
		if s.Plots[0].Genetics != 3 || s.Plots[0].Nursery != 5 {
			t.Fatal("later upgrades changed a planted crop")
		}
		calls := 0
		roll := func(int) (int, error) {
			calls++
			if calls == 5 {
				return recovery, nil
			}
			return 99, nil
		}
		if _, _, _, err := apply(&s, Action{Kind: "harvest"}, now.Add(GrowTime), roll); err != nil {
			t.Fatal(err)
		}
		want := 3
		if recovery == 49 {
			want++
		}
		if s.Seeds["carrot"] != want || calls != 5 {
			t.Fatal("nursery did not return exactly one seed at its boundary")
		}
		if _, _, _, err := apply(&s, Action{Kind: "harvest"}, now.Add(GrowTime), roll); err == nil || s.Seeds["carrot"] != want {
			t.Fatal("repeat harvest duplicated seeds")
		}
	}
}

func TestExpandedGalleryAndUpgradeLimits(t *testing.T) {
	now := time.Now()
	s := initialState(now)
	s.Coins = 1000000
	s.XP = 100000
	s.Barn = []Harvest{{ID: "batch", Quantity: 13, Value: 1300}}
	for level := 0; level < 3; level++ {
		if _, _, _, err := apply(&s, Action{Kind: "upgrade", Upgrade: "gallery"}, now, fixed); err != nil {
			t.Fatal(err)
		}
		if showcaseCapacity(s) != 8+level*2 {
			t.Fatal("wrong gallery capacity")
		}
	}
	for i := 0; i < 12; i++ {
		if _, _, _, err := apply(&s, Action{Kind: "keep", Item: "batch"}, now, fixed); err != nil {
			t.Fatal(err)
		}
	}
	if _, _, _, err := apply(&s, Action{Kind: "keep", Item: "batch"}, now, fixed); err == nil || s.Barn[0].Quantity != 1 {
		t.Fatal("expanded gallery exceeded its capacity")
	}
	for _, upgrade := range []string{"genetics", "nursery", "gallery"} {
		for _, meta := range upgradesFor(s) {
			if meta.ID == upgrade {
				s.Upgrades[upgrade] = meta.MaxLevel
			}
		}
		coins := s.Coins
		if _, _, _, err := apply(&s, Action{Kind: "upgrade", Upgrade: upgrade}, now, fixed); err == nil || s.Coins != coins {
			t.Fatalf("%s cap ignored", upgrade)
		}
	}
}
