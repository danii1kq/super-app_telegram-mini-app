package farm

import (
	"testing"
	"time"
)

func TestBreedingConsumesOnlyDuplicatesAndClaimsOnce(t *testing.T) {
	now := time.Now()
	s := initialState(now)
	coins := s.Coins
	if _, _, _, err := apply(&s, Action{Kind: "breed", Seed: "strawberry"}, now, fixed); err == nil {
		t.Fatal("accepted only one seed")
	}
	if _, _, _, err := apply(&s, Action{Kind: "breed", Seed: "carrot"}, now, fixed); err != nil {
		t.Fatal(err)
	}
	if s.Seeds["carrot"] != 2 || s.Coins != coins || s.Breeding.Result != "aurora_root" {
		t.Fatal("wrong breeding cost or recipe")
	}
	if _, _, _, err := apply(&s, Action{Kind: "breed", Seed: "carrot"}, now, fixed); err == nil {
		t.Fatal("parallel job accepted")
	}
	if _, _, _, err := apply(&s, Action{Kind: "claim_breed"}, now.Add(GrowTime-time.Second), fixed); err == nil {
		t.Fatal("early breeding claim")
	}
	if _, _, _, err := apply(&s, Action{Kind: "claim_breed"}, now.Add(GrowTime), fixed); err != nil {
		t.Fatal(err)
	}
	if s.Seeds["aurora_root"] != 1 || s.Breeding != nil || len(s.Hybrids) != 1 {
		t.Fatal("hybrid missing")
	}
	if _, _, _, err := apply(&s, Action{Kind: "claim_breed"}, now.Add(GrowTime), fixed); err == nil {
		t.Fatal("double claim accepted")
	}
	if _, _, _, err := apply(&s, Action{Kind: "buy", Seed: "aurora_root"}, now, fixed); err == nil {
		t.Fatal("hybrid bought for free")
	}
	if _, _, _, err := apply(&s, Action{Kind: "plant", Seed: "aurora_root", Slot: 2, Care: "water"}, now, fixed); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := apply(&s, Action{Kind: "harvest"}, now.Add(GrowTime), fixed); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, item := range s.Barn {
		if item.Seed == "aurora_root" {
			found = true
			if item.Value <= 0 || item.ID == "" || item.Weight <= 0 {
				t.Fatal("invalid hybrid harvest")
			}
		}
	}
	if !found {
		t.Fatal("hybrid not harvestable")
	}
}

func TestShowcaseConservesHarvestAndProtectsFromSale(t *testing.T) {
	now := time.Now()
	s := initialState(now)
	s.Barn = []Harvest{{Seed: "pumpkin", Quality: "perfect", Mutation: "gold", Quantity: 3, Value: 1800, Weight: 7000}}
	normalizeState(&s)
	source := s.Barn[0].ID
	if _, _, _, err := apply(&s, Action{Kind: "keep", Item: source}, now, fixed); err != nil {
		t.Fatal(err)
	}
	if s.Barn[0].Quantity != 2 || s.Barn[0].Value != 1200 || len(s.Showcase) != 1 || s.Showcase[0].Plant.Value != 600 {
		t.Fatal("trophy duplicated or destroyed value")
	}
	trophy := s.Showcase[0].Plant
	if trophy.ID == source || trophy.Quantity != 1 || trophy.Weight != 7000 {
		t.Fatal("wrong trophy identity")
	}
	_, _, sold, err := apply(&s, Action{Kind: "sell"}, now, fixed)
	if err != nil || sold != 1200 || len(s.Showcase) != 1 {
		t.Fatal("sale touched showcase")
	}
	if _, _, _, err = apply(&s, Action{Kind: "return_trophy", Item: trophy.ID}, now, fixed); err != nil {
		t.Fatal(err)
	}
	if len(s.Showcase) != 0 || len(s.Barn) != 1 || s.Barn[0].Value != 600 {
		t.Fatal("return lost trophy")
	}
	if _, _, _, err = apply(&s, Action{Kind: "return_trophy", Item: trophy.ID}, now, fixed); err == nil {
		t.Fatal("double trophy return accepted")
	}
}

func TestRecipesCoverBaseSeedsAndShowcaseLimit(t *testing.T) {
	for _, crop := range Crops {
		recipe, ok := recipeFor(crop.ID)
		if !ok || recipe.Crop.SeedPrice != 0 || recipe.Crop.Price <= crop.SeedPrice {
			t.Fatalf("missing recipe: %s", crop.ID)
		}
	}
	s := initialState(time.Now())
	s.Barn = []Harvest{{ID: "batch", Quantity: 8, Value: 800}}
	for i := 0; i < 6; i++ {
		if _, _, _, err := apply(&s, Action{Kind: "keep", Item: "batch"}, time.Now(), fixed); err != nil {
			t.Fatal(err)
		}
	}
	if _, _, _, err := apply(&s, Action{Kind: "keep", Item: "batch"}, time.Now(), fixed); err == nil || s.Barn[0].Quantity != 2 {
		t.Fatal("showcase limit lost plants")
	}
}
