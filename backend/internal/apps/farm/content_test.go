package farm

import (
	"testing"
	"time"
)

func TestWonderGardenSpeciesAndHybridsArePlayable(t *testing.T) {
	now := time.Now()
	want := map[string]string{
		"pepper": "flame_pepper", "lavender": "flame_pepper",
		"pear": "honey_pear", "mango": "honey_pear",
		"lotus": "astral_lotus", "starfruit": "astral_lotus",
	}
	for seed, hybrid := range want {
		s := initialState(now)
		s.XP = 100000
		s.Plots = make([]*Plot, 4)
		s.Coins = 10000
		crop, ok := species(seed)
		if !ok {
			t.Fatalf("missing new species %s", seed)
		}
		if _, _, _, err := apply(&s, Action{Kind: "buy", Seed: seed}, now, fixed); err != nil {
			t.Fatal(err)
		}
		if s.Coins != 10000-crop.SeedPrice || s.Seeds[seed] != 1 {
			t.Fatalf("wrong shop value for %s", seed)
		}
		if _, _, _, err := apply(&s, Action{Kind: "plant", Seed: seed, Slot: 0, Care: "water"}, now, fixed); err != nil {
			t.Fatal(err)
		}
		if _, _, _, err := apply(&s, Action{Kind: "harvest"}, now.Add(GrowTime), fixed); err != nil {
			t.Fatal(err)
		}
		if len(s.Barn) != 1 || s.Barn[0].Seed != seed || s.Barn[0].Value != crop.Price || s.Barn[0].Weight <= 0 {
			t.Fatalf("new crop is not harvestable: %+v", s.Barn)
		}
		s.Seeds[seed] = 2
		if _, _, _, err := apply(&s, Action{Kind: "breed", Seed: seed}, now, fixed); err != nil {
			t.Fatal(err)
		}
		if _, _, _, err := apply(&s, Action{Kind: "claim_breed"}, now.Add(GrowTime), fixed); err != nil {
			t.Fatal(err)
		}
		if s.Seeds[hybrid] != 1 {
			t.Fatalf("%s did not produce %s", seed, hybrid)
		}
		if _, _, _, err := apply(&s, Action{Kind: "plant", Seed: hybrid, Slot: 0, Care: "water"}, now, fixed); err != nil {
			t.Fatal(err)
		}
		if _, _, _, err := apply(&s, Action{Kind: "harvest"}, now.Add(GrowTime), fixed); err != nil {
			t.Fatal(err)
		}
		if s.Barn[1].Seed != hybrid || s.Barn[1].Value <= crop.SeedPrice {
			t.Fatalf("bad hybrid harvest: %+v", s.Barn[1])
		}
	}
}

func TestExpandedContentPreservesOriginalPackOdds(t *testing.T) {
	original := [][]int{
		{20, 18, 12, 8, 5, 1, 12, 9, 6, 5, 3, 1},
		{3, 5, 10, 12, 16, 5, 4, 6, 12, 10, 12, 5},
		{0, 0, 5, 5, 25, 15, 0, 0, 5, 5, 25, 15},
	}
	for p, weights := range original {
		for i, weight := range weights {
			if Packs[p].Weights[i] != weight {
				t.Fatalf("changed old pack %s", Packs[p].ID)
			}
		}
		for level := 0; level <= 5; level++ {
			for i, weight := range packsFor(level)[p].Weights {
				if i >= 12 && weight != 0 {
					t.Fatalf("new species leaked into old pack %s", Packs[p].ID)
				}
			}
		}
	}
	s := initialState(time.Now())
	normalizeState(&s)
	if s.Pity["orchard"] != 0 || len(s.Plots) != 4 || len(s.Seeds) != 3 {
		t.Fatal("new content changed the starter save")
	}
}

func TestOrchardPackGuaranteeUsesItsOwnCounter(t *testing.T) {
	now := time.Now()
	s := initialState(now)
	s.Coins = 1100
	s.XP = 100000
	s.Pity["orchard"] = 9
	_, drops, _, err := apply(&s, Action{Kind: "pack", Pack: "orchard"}, now, func(int) (int, error) { return 15, nil })
	if err != nil || len(drops) != 3 || drops[0] != "blueberry" || drops[1] != "pepper" || s.Pity["orchard"] != 2 || s.Pity["garden"] != 0 || s.Coins != 0 {
		t.Fatalf("orchard guarantee failed: %v %+v %v", drops, s.Pity, err)
	}
}
