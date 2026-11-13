package farm

import (
	"fmt"
	"math/rand"
	"testing"
	"time"
)

type progressionRun struct {
	firstUpgrade, level10, level20, level40, level50, completed int
	level, coins, sold, upgrades                                int
}

// Models ordinary farming with daily rewards and expeditions, without orders,
// packs, minigames or developer grants. Days are virtual; no live save is used.
func TestProgressionAcrossRegularVisits(t *testing.T) {
	for _, visits := range []int{2, 4} {
		t.Run(fmt.Sprintf("%d_visits_per_day", visits), func(t *testing.T) {
			for seed := int64(1); seed <= 10; seed++ {
				got := simulateProgression(t, visits, seed)
				t.Logf("seed=%d first_upgrade_day=%d level10_day=%d level20_day=%d level40_day=%d level50_day=%d all_upgrades_day=%d final_level=%d upgrades=%d coins=%d sold=%d", seed, got.firstUpgrade, got.level10, got.level20, got.level40, got.level50, got.completed, got.level, got.upgrades, got.coins, got.sold)
				if got.firstUpgrade == 0 || got.firstUpgrade > 7 || got.level20 == 0 || got.level < 20 {
					t.Fatalf("seed=%d: regular farming stalled: %+v", seed, got)
				}
				if got.completed == 1 || got.level50 == 1 {
					t.Fatalf("seed=%d: progression finished on the first day: %+v", seed, got)
				}
				if got.level10 > 25 || got.level20 > 70 {
					t.Fatalf("seed=%d: early progression became too slow: %+v", seed, got)
				}
				if got.completed != 0 && got.completed < 300/visits {
					t.Fatalf("seed=%d: late progression ran out too early: %+v", seed, got)
				}
				if visits == 4 && (got.completed == 0 || got.level50 == 0) {
					t.Fatalf("seed=%d: regular play cannot finish within 180 days: %+v", seed, got)
				}
			}
		})
	}
}

func simulateProgression(t *testing.T, visits int, seed int64) progressionRun {
	t.Helper()
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	s := initialState(start)
	rng := rand.New(rand.NewSource(seed))
	roll := func(n int) (int, error) { return rng.Intn(n), nil }
	result := progressionRun{}
	priority := []string{"land", "soil", "water", "market", "genetics", "lamp", "nursery", "selection", "gallery"}
	for visit := 0; visit < 180*visits; visit++ {
		now := start.Add(time.Duration(visit) * (24 * time.Hour / time.Duration(visits)))
		day := visit/visits + 1
		act := func(action Action) {
			t.Helper()
			if _, _, _, err := apply(&s, action, now, roll); err != nil {
				t.Fatalf("seed=%d visits=%d day=%d action=%+v: %v", seed, visits, day, action, err)
			}
			if s.Coins < 0 {
				t.Fatalf("seed=%d day=%d: negative coins after %+v", seed, day, action)
			}
		}
		for _, plot := range s.Plots {
			if plot != nil && !plot.ReadyAt.After(now) {
				act(Action{Kind: "harvest"})
				break
			}
		}
		if s.Expedition != nil && !s.Expedition.ReadyAt.After(now) {
			act(Action{Kind: "claim_expedition"})
		}
		if _, _, claimable := dailyState(s, now); claimable {
			act(Action{Kind: "claim_daily"})
		}
		for _, level := range claimableLevels(s) {
			act(Action{Kind: "claim_level", Level: level})
		}
		if len(s.Barn) > 0 {
			act(Action{Kind: "sell"})
		}
		level := levelFor(s.XP)
		for _, milestone := range []struct {
			level int
			day   *int
		}{{10, &result.level10}, {20, &result.level20}, {40, &result.level40}, {50, &result.level50}} {
			if level >= milestone.level && *milestone.day == 0 {
				*milestone.day = day
			}
		}
		crop := simulationCrop(s)
		_, careCost := simulationCare(s, crop)
		// Reserve a full planting cycle, including a possible new bed.
		reserve := (len(s.Plots) + 1) * (crop.SeedPrice + careCost)
		for {
			bought := false
			for _, id := range priority {
				for _, upgrade := range upgradesFor(s) {
					if upgrade.ID != id || upgrade.Level >= upgrade.MaxLevel || level < upgrade.RequiredLevel || s.Coins-upgrade.Cost < reserve {
						continue
					}
					act(Action{Kind: "upgrade", Upgrade: id})
					if result.firstUpgrade == 0 {
						result.firstUpgrade = day
					}
					result.upgrades++
					bought = true
				}
			}
			if !bought {
				break
			}
		}
		complete := true
		for _, upgrade := range Upgrades {
			if s.Upgrades[upgrade.ID] < upgrade.MaxLevel {
				complete = false
			}
			if upgrade.ID != "land" && s.Upgrades[upgrade.ID] >= upgrade.MaxLevel && level < upgradeRequiredLevel(upgrade.MaxLevel) {
				t.Fatalf("seed=%d: maxed %s before its required player level", seed, upgrade.ID)
			}
		}
		if complete && result.completed == 0 {
			result.completed = day
		}
		crop = simulationCrop(s)
		for needed := max(0, len(s.Plots)-s.Seeds[crop.ID]); needed > 0; needed-- {
			act(Action{Kind: "buy", Seed: crop.ID})
		}
		for slot, plot := range s.Plots {
			if plot != nil {
				continue
			}
			chosen := crop
			for _, owned := range Crops {
				if s.Seeds[owned.ID] > 0 && owned.Price > chosen.Price {
					chosen = owned
				}
			}
			care, cost := simulationCare(s, chosen)
			if s.Coins < cost {
				care = "water"
			}
			act(Action{Kind: "plant", Slot: slot, Seed: chosen.ID, Care: care})
		}
		if s.Expedition == nil {
			for i := len(expeditionRoutes) - 1; i >= 0; i-- {
				route := expeditionRoutes[i]
				if level >= route.RequiredLevel && s.Coins-route.Price >= reserve {
					act(Action{Kind: "start_expedition", Item: route.ID})
					break
				}
			}
		}
	}
	result.level, result.coins, result.sold = levelFor(s.XP), s.Coins, s.Sold
	if len(s.Plots) != 8 || len(s.GameHistory) != 0 {
		t.Fatal("simulation failed to build the garden or used a minigame")
	}
	return result
}

func simulationCrop(s State) Species {
	best := Crops[0]
	for _, crop := range Crops {
		_, careCost := simulationCare(s, crop)
		cost := max(0, len(s.Plots)-s.Seeds[crop.ID])*crop.SeedPrice + len(s.Plots)*careCost
		if levelFor(s.XP) >= seedUnlockLevel(crop.ID) && crop.Price > best.Price && cost <= s.Coins {
			best = crop
		}
	}
	return best
}

func simulationCare(s State, crop Species) (string, int) {
	quality := 1 + float64(2*perfectPercent(s.Upgrades["water"])+choicePercent(s.Upgrades["water"]))/100
	yield := 1 + float64(soilPercent(s.Upgrades["soil"]))/100
	mutation := 0.0
	for roll := 0; roll < 100; roll++ {
		_, multiplier := chooseMutation(s.Upgrades["genetics"], roll)
		mutation += float64(multiplier) / 100
	}
	mutation = 1 + float64(mutationBP(s.Upgrades["lamp"], s.Upgrades["genetics"]))/10000*(mutation-1)
	sale := float64(crop.Price) * mutation * float64(100+marketPercent(s.Upgrades["market"])) / 100
	if sale*(quality*(yield+.25)-(quality+.20)*yield) > 12 {
		return "soil", 12
	}
	return "water", 0
}
