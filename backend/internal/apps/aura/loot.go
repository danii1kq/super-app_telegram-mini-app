package aura

import "math/rand"

type lootWeight struct {
	rank   Rank
	weight int
}

var cryptLoot = []lootWeight{{RankE, 6000}, {RankD, 3000}, {RankC, 1000}}
var forgeLoot = []lootWeight{{RankE, 4500}, {RankD, 3000}, {RankC, 1800}, {RankB, 600}, {RankA, 100}}
var voidLoot = []lootWeight{{RankE, 4000}, {RankD, 3000}, {RankC, 1800}, {RankB, 800}, {RankA, 300}, {RankS, 100}}
var falseGateLoot = []lootWeight{{RankB, 6500}, {RankA, 3000}, {RankS, 450}, {RankSS, 48}, {RankSSS, 2}}
var redGateLoot = []lootWeight{{RankB, 8000}, {RankA, 1800}, {RankS, 190}, {RankSS, 10}}
var shrineLoot = []lootWeight{{RankB, 9000}, {RankA, 950}, {RankS, 50}}

func ordinaryLoot(session *Session) (int, []lootWeight) {
	switch session.Kind {
	case SessionGate:
		switch session.Dungeon {
		case "forge":
			return 45, forgeLoot
		case "void":
			return 55, voidLoot
		default:
			return 35, cryptLoot
		}
	case SessionArena:
		return 15, cryptLoot
	case SessionTower:
		chance := 25
		if session.Floor%5 == 0 {
			chance = 45
		}
		if session.Floor >= 25 {
			return chance, voidLoot
		}
		if session.Floor >= 10 {
			return chance, forgeLoot
		}
		return chance, cryptLoot
	}
	return 0, nil
}

func eventLootWeights(id string) []lootWeight {
	switch id {
	case "false_gate":
		return falseGateLoot
	case "red_gate":
		return redGateLoot
	case "hidden_shrine":
		return shrineLoot
	}
	return nil
}

func lootRank(weights []lootWeight, roll int) Rank {
	for _, entry := range weights {
		if roll < entry.weight {
			return entry.rank
		}
		roll -= entry.weight
	}
	return RankE
}

func eligibleEventRank(rank Rank, session *Session, progress *Progress) Rank {
	if atLeast(rank, RankSS) && (session.Dungeon != "void" || progress.Clears < 20) {
		return RankB
	}
	if rank == RankSSS && progress.Clears < 80 {
		return RankB
	}
	return rank
}

// Eligibility replaces locked top ranks with B; their weight never boosts another rare tier.
func rollLoot(rnd *rand.Rand, session *Session, progress *Progress) *Relic {
	if session.BossHP > 0 || session.HeroHP <= 0 || session.Kind == SessionGrind {
		return nil
	}
	if session.Event != nil && session.Event.Accepted && session.Kind == SessionGate {
		weights := eventLootWeights(session.Event.ID)
		if len(weights) == 0 {
			return nil
		}
		rank := eligibleEventRank(lootRank(weights, rnd.Intn(10000)), session, progress)
		if session.Event.ID == "false_gate" && session.Dungeon == "void" && progress.Clears >= 20 {
			progress.SSEventPity++
			if progress.Clears >= 80 {
				progress.SSSEventPity++
			}
			if progress.SSSEventPity >= 20 {
				rank = RankSSS
			} else if progress.SSEventPity >= 5 && !atLeast(rank, RankSS) {
				rank = RankSS
			}
			if atLeast(rank, RankSS) {
				progress.SSEventPity = 0
			}
			if rank == RankSSS {
				progress.SSSEventPity = 0
			}
		}
		progress.Pity = 0
		return pickLootRelic(rnd, rank, session.Event.ID, session.Dungeon)
	}
	chance, weights := ordinaryLoot(session)
	if chance == 0 {
		return nil
	}
	if progress.Pity < RelicPity-1 && rnd.Intn(10000) >= chance*100 {
		progress.Pity++
		return nil
	}
	progress.Pity = 0
	rank := lootRank(weights, rnd.Intn(10000))
	return pickLootRelic(rnd, rank, "dungeon", session.Dungeon)
}

func pickLootRelic(rnd *rand.Rand, rank Rank, source, dungeon string) *Relic {
	for tier := rankIndex(rank); tier >= 0; tier-- {
		pool := []Relic{}
		for _, item := range relicCatalog {
			item = relicMetadata(item)
			if item.Rank == rankOrder[tier] && item.Source == source && (item.Dungeon == "" || item.Dungeon == dungeon || dungeon == "") {
				pool = append(pool, item)
			}
		}
		if len(pool) > 0 {
			item := pool[rnd.Intn(len(pool))]
			return &item
		}
	}
	return nil
}

func lootRules() map[string]any {
	percent := func(weights []lootWeight) map[string]float64 {
		result := map[string]float64{}
		for _, entry := range weights {
			result[string(entry.rank)] = float64(entry.weight) / 100
		}
		return result
	}
	return map[string]any{
		"pity_after":     RelicPity,
		"pity_guarantee": "item",
		"dungeons": map[string]any{
			"crypt": map[string]any{"chance_percent": 35, "max_rank": "C", "rank_percent": percent(cryptLoot)},
			"forge": map[string]any{"chance_percent": 45, "max_rank": "A", "rank_percent": percent(forgeLoot)},
			"void":  map[string]any{"chance_percent": 55, "max_rank": "S", "rank_percent": percent(voidLoot)},
		},
		"arena_percent": 15, "tower_percent": 25, "tower_fifth_percent": 45,
		"events":        map[string]any{"false_gate": percent(falseGateLoot), "red_gate": percent(redGateLoot), "hidden_shrine": percent(shrineLoot)},
		"ss_min_clears": 20, "sss_min_clears": 80, "top_rank_dungeon": "void",
		"ss_event_pity": 5, "sss_event_pity": 20,
	}
}
