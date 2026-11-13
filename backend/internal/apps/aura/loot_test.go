package aura

import "testing"

func TestExclusiveGuaranteeCountsOnlyEligibleVictories(t *testing.T) {
	rnd := seeded(72)
	session := &Session{Kind: SessionGate, Dungeon: "void", HeroHP: 100, Event: &DungeonEvent{ID: "false_gate", Accepted: true}}
	progress := &Progress{Clears: 80, SSEventPity: 4, SSSEventPity: 19}
	item := rollLoot(rnd, session, progress)
	if item == nil || item.Rank != RankSSS || progress.SSSEventPity != 0 || progress.SSEventPity != 0 {
		t.Fatal("final exclusive guarantee failed", item)
	}
	progress = &Progress{Clears: 20, SSEventPity: 4}
	item = rollLoot(rnd, session, progress)
	if item == nil || !atLeast(item.Rank, RankSS) || progress.SSEventPity != 0 || progress.SSSEventPity != 0 {
		t.Fatal("SS guarantee failed", item)
	}
	session.HeroHP = 0
	progress.SSEventPity = 4
	if rollLoot(rnd, session, progress) != nil || progress.SSEventPity != 4 {
		t.Fatal("failed battle advanced exclusive guarantee")
	}
}

func TestExpandedRelicsKeepLegacyAndHonestCraftMetadata(t *testing.T) {
	if len(relicCatalog) != 64 {
		t.Fatal("expected 64 relics", len(relicCatalog))
	}
	events, craftable := 0, 0
	for _, item := range relicsCatalog() {
		if item.Source != "dungeon" {
			events++
		}
		if item.Craftable {
			craftable++
			if item.CraftAura <= 0 || len(item.CraftCost) == 0 || atLeast(item.Rank, RankS) || item.Source != "dungeon" {
				t.Fatal("invalid craftable relic", item)
			}
		} else if item.CraftAura != 0 || len(item.CraftCost) != 0 {
			t.Fatal("exclusive relic advertises a recipe", item)
		}
		if item.Description == "" || item.Value <= 0 {
			t.Fatal("missing real effect", item.Key)
		}
		if upgraded := scaledRelic(item, 3); len(upgraded.UpgradeMaterials) == 0 {
			t.Fatal("late awakening has no material cost", item.Key)
		}
	}
	if events != 18 || craftable != 43 {
		t.Fatal("exclusive or craftable counts drifted", events, craftable)
	}
	for _, key := range []string{"rusty_blade", "world_ender", "world_shell", "origin_crown", "echo_blade", "troll_heart", "hourglass", "executioner"} {
		if _, ok := relicByKey(key); !ok {
			t.Fatal("legacy relic removed", key)
		}
	}
}

func TestOrdinaryLootCannotProduceEventOrAboveDungeonCap(t *testing.T) {
	rnd := seeded(104)
	for dungeon, cap := range map[string]Rank{"crypt": RankC, "forge": RankA, "void": RankS} {
		session := &Session{Kind: SessionGate, Dungeon: dungeon, HeroHP: 100}
		progress := &Progress{Clears: 1000}
		for i := 0; i < 20000; i++ {
			progress.Pity = 9
			item := rollLoot(rnd, session, progress)
			if item == nil || item.Source != "dungeon" || rankIndex(item.Rank) > rankIndex(cap) {
				t.Fatal("ordinary loot bypassed cap/source", dungeon, item)
			}
		}
	}
	for i := 0; i < 1000; i++ {
		if item := rollRelic(rnd, RankSSS); relicMetadata(item).Source != "dungeon" || atLeast(item.Rank, RankSS) {
			t.Fatal("legacy picker leaked exclusive", item)
		}
	}
}

func TestLootPityGuaranteesItemWithoutChangingRankTable(t *testing.T) {
	rnd := seeded(15)
	session := &Session{Kind: SessionGate, Dungeon: "crypt", HeroHP: 100}
	progress := &Progress{Pity: 9}
	if item := rollLoot(rnd, session, progress); item == nil || progress.Pity != 0 || atLeast(item.Rank, RankB) {
		t.Fatal("pity promised wrong result", item, progress.Pity)
	}
	misses := 0
	for i := 0; i < 10000; i++ {
		if item := rollLoot(rnd, session, progress); item == nil {
			misses++
		} else {
			misses = 0
		}
		if misses >= 10 {
			t.Fatal("ten consecutive empty rewarded victories")
		}
	}
	progress.Pity = 9
	session.Kind = SessionGrind
	if item := rollLoot(rnd, session, progress); item != nil || progress.Pity != 9 {
		t.Fatal("grind changed relic pity")
	}
}

func TestEventLootEligibilityAndExactRankBoundaries(t *testing.T) {
	if lootRank(falseGateLoot, 9499) != RankA || lootRank(falseGateLoot, 9500) != RankS || lootRank(falseGateLoot, 9950) != RankSS || lootRank(falseGateLoot, 9998) != RankSSS {
		t.Fatal("false gate boundaries changed")
	}
	session := &Session{Kind: SessionGate, Dungeon: "void", HeroHP: 100, Event: &DungeonEvent{ID: "false_gate", Accepted: true}}
	progress := &Progress{Clears: 19}
	if eligibleEventRank(RankSS, session, progress) != RankB {
		t.Fatal("early SS unlocked")
	}
	progress.Clears = 20
	if eligibleEventRank(RankSS, session, progress) != RankSS || eligibleEventRank(RankSSS, session, progress) != RankB {
		t.Fatal("SS boundary")
	}
	progress.Clears = 80
	if eligibleEventRank(RankSSS, session, progress) != RankSSS {
		t.Fatal("SSS boundary")
	}
	session.Dungeon = "crypt"
	if eligibleEventRank(RankSSS, session, progress) != RankB {
		t.Fatal("starter dungeon bypassed late loot")
	}
	rnd := seeded(16)
	for _, id := range []string{"false_gate", "red_gate", "hidden_shrine"} {
		session.Event.ID = id
		for i := 0; i < 1000; i++ {
			item := rollLoot(rnd, session, progress)
			if item == nil || item.Source != id || atLeast(item.Rank, RankSS) {
				t.Fatal("wrong event source or early top rank", id, item)
			}
		}
	}
	session.Event.Accepted = false
	progress.Pity = 9
	if item := rollLoot(rnd, session, progress); item == nil || item.Source != "dungeon" {
		t.Fatal("unaccepted anomaly gave exclusive loot")
	}
}

func TestLootBaseFrequenciesAndRareSTier(t *testing.T) {
	const runs = 100000
	for _, dungeon := range []string{"crypt", "forge", "void"} {
		rnd := seeded(66)
		drops, sTier := 0, 0
		session := &Session{Kind: SessionGate, Dungeon: dungeon, HeroHP: 100}
		progress := &Progress{}
		for i := 0; i < runs; i++ {
			progress.Pity = 0
			if item := rollLoot(rnd, session, progress); item != nil {
				drops++
				if item.Rank == RankS {
					sTier++
				}
			}
		}
		chance, _ := ordinaryLoot(session)
		if drops < runs*(chance-1)/100 || drops > runs*(chance+1)/100 {
			t.Fatal("drop frequency outside base chance", dungeon, drops)
		}
		if dungeon == "void" && (sTier < drops*6/1000 || sTier > drops*15/1000) {
			t.Fatal("void S tier is not approximately one percent of drops", sTier, drops)
		}
		t.Logf("%s: %d relics, %d S in %d runs", dungeon, drops, sTier, runs)
	}
	rnd := seeded(67)
	session := &Session{Kind: SessionGate, Dungeon: "void", HeroHP: 100, Event: &DungeonEvent{ID: "false_gate", Accepted: true}}
	progress := &Progress{Clears: 80}
	sTier, ssTier, sssTier := 0, 0, 0
	for i := 0; i < runs; i++ {
		// Measure the advertised base roll; accumulated guarantees are checked separately.
		progress.SSEventPity, progress.SSSEventPity = 0, 0
		item := rollLoot(rnd, session, progress)
		switch item.Rank {
		case RankS:
			sTier++
		case RankSS:
			ssTier++
		case RankSSS:
			sssTier++
		}
	}
	if sTier < 4000 || sTier > 5000 || ssTier < 350 || ssTier > 650 || sssTier < 3 || sssTier > 50 {
		t.Fatal("event top ranks are too common or inaccessible", sTier, ssTier, sssTier)
	}
	t.Logf("eligible S-event: S=%d SS=%d SSS=%d in %d victories", sTier, ssTier, sssTier, runs)
}
