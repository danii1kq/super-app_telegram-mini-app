package farm

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

func TestExpeditionCostsAndRewardsExactlyOnce(t *testing.T) {
	now := time.Date(2026, 11, 13, 8, 0, 0, 0, time.UTC)
	for _, route := range expeditionCatalog() {
		t.Run(route.ID, func(t *testing.T) {
			s := initialState(now)
			s.Coins = route.Price
			before := copyExpeditionSeeds(s.Seeds)
			if _, err := expeditionAction(&s, Action{Kind: "start_expedition", Item: route.ID}, now); err != nil {
				t.Fatal(err)
			}
			if s.Coins != 0 || s.Expedition.Route != route.ID || !s.Expedition.ReadyAt.Equal(now.Add(GrowTime)) {
				t.Fatal("wrong start price, route or duration")
			}
			if _, err := expeditionAction(&s, Action{Kind: "claim_expedition"}, now.Add(GrowTime-time.Nanosecond)); err == nil {
				t.Fatal("claimed before the server deadline")
			}
			if !reflect.DeepEqual(s.Seeds, before) || s.XP != 0 {
				t.Fatal("early claim changed rewards")
			}
			if _, err := expeditionAction(&s, Action{Kind: "claim_expedition"}, now.Add(GrowTime)); err != nil {
				t.Fatal(err)
			}
			for id, count := range route.Seeds {
				if s.Seeds[id] != before[id]+count {
					t.Fatalf("wrong reward for %s", id)
				}
			}
			if s.XP != route.XP || s.Expedition != nil || s.Coins != 0 {
				t.Fatal("wrong experience, cleared job or currency")
			}
			saved, _ := json.Marshal(s)
			if _, err := expeditionAction(&s, Action{Kind: "claim_expedition"}, now.Add(GrowTime)); err == nil {
				t.Fatal("claimed twice")
			}
			after, _ := json.Marshal(s)
			if string(saved) != string(after) {
				t.Fatal("repeated claim changed the save")
			}
		})
	}
}

func TestExpeditionRejectsInvalidStartsWithoutSpending(t *testing.T) {
	now := time.Now()
	s := initialState(now)
	s.Coins = 89
	for _, action := range []Action{{Kind: "start_expedition", Item: "meadow"}, {Kind: "start_expedition", Item: "missing"}, {Kind: "claim_expedition"}, {Kind: "missing"}} {
		if _, err := expeditionAction(&s, action, now); err == nil || s.Coins != 89 || s.Expedition != nil {
			t.Fatalf("invalid action changed the save: %+v", action)
		}
	}
	s.Coins = 1000
	if _, err := expeditionAction(&s, Action{Kind: "start_expedition", Item: "meadow"}, now); err != nil {
		t.Fatal(err)
	}
	job := s.Expedition
	for _, when := range []time.Time{now, now.Add(GrowTime)} {
		if _, err := expeditionAction(&s, Action{Kind: "start_expedition", Item: "orchard"}, when); err == nil || s.Coins != 910 || s.Expedition != job {
			t.Fatal("an ongoing or unclaimed expedition was replaced")
		}
	}
}

func TestExpeditionSnapshotsAndCatalogStayIndependent(t *testing.T) {
	now := time.Now()
	s := initialState(now)
	if _, err := expeditionAction(&s, Action{Kind: "start_expedition", Item: "meadow"}, now); err != nil {
		t.Fatal(err)
	}
	catalog := expeditionCatalog()
	catalog[0].Seeds["pepper"] = 999
	catalog[0].Price = 0
	if expeditionCatalog()[0].Seeds["pepper"] != 2 || expeditionCatalog()[0].Price != 90 || s.Expedition.Seeds["pepper"] != 2 {
		t.Fatal("returned metadata changed the server catalog or job")
	}
	s.Expedition.Seeds["pepper"] = 3
	if expeditionCatalog()[0].Seeds["pepper"] != 2 {
		t.Fatal("job shares the reward map with the server catalog")
	}
	data, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	var restored State
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatal(err)
	}
	if _, err := expeditionAction(&restored, Action{Kind: "claim_expedition"}, now.Add(GrowTime)); err != nil || restored.Seeds["pepper"] != 3 {
		t.Fatal("saved promised rewards did not survive reload")
	}
}
