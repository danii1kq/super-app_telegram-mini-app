package aura

import (
	"strings"
	"testing"
	"time"
)

func TestTwoPieceSetsHaveRealEffectsAndRequireBothItems(t *testing.T) {
	for _, set := range relicSets {
		one, _ := relicByKey(set.Keys[0])
		two, _ := relicByKey(set.Keys[1])
		if len(activeSetIDs([]Relic{one})) != 0 || len(activeSetIDs([]Relic{one, one})) != 0 {
			t.Fatal("set active from one or duplicate item", set.ID)
		}
		ids := activeSetIDs([]Relic{one, two})
		if len(ids) != 1 || ids[0] != set.ID {
			t.Fatal("two-piece set did not activate", set.ID)
		}
	}
	base := Session{Kind: SessionGate, Room: 2, Rooms: 3, Round: 1, Combo: 1}
	crypt := base
	crypt.Sets = []string{"crypt"}
	if calculateHit(&crypt, 1) <= calculateHit(&base, 1) {
		t.Fatal("crypt set has no damage bonus")
	}
	forge := base
	forge.Sets = []string{"forge"}
	if incomingHit(&forge, 1) >= incomingHit(&base, 1) {
		t.Fatal("forge set has no protection bonus")
	}
	void := base
	void.Sets = []string{"void"}
	if calculateHit(&void, .05) != 2*calculateHit(&base, .05) {
		t.Fatal("void set has no crit bonus")
	}
	snapshot := Session{Sets: activeSetIDs([]Relic{{Key: "bone_rapier"}, {Key: "abyss_scale"}})}
	_ = activeSetIDs([]Relic{{Key: "ash_halberd"}, {Key: "ember_carapace"}})
	if !hasSet(&snapshot, "crypt") || hasSet(&snapshot, "forge") {
		t.Fatal("new equipment changed old run snapshot")
	}
}
func TestCursedRelicsOfferRealTradeoffs(t *testing.T) {
	for _, key := range []string{"curse_fang", "curse_eye", "curse_codex"} {
		item, ok := relicByKey(key)
		if !ok || item.Curse == "" || item.Value <= 0 || atLeast(item.Rank, RankS) {
			t.Fatal("invalid cursed item", key)
		}
	}
	fang, _ := relicByKey("curse_fang")
	session := Session{Worn: []Relic{fang}, Charge: 3, AbilityReady: true}
	curseMistake(&session)
	if session.Charge != 0 || session.AbilityReady {
		t.Fatal("fang curse does not drain charge")
	}
	eye, _ := relicByKey("curse_eye")
	session.Worn = []Relic{eye}
	session.HeroMaxHP = 120
	if curseAbilityCost(&session) != 12 {
		t.Fatal("eye curse costs no HP")
	}
	codex, _ := relicByKey("curse_codex")
	session.Worn = []Relic{codex}
	session.HeroHP = 40
	session.HeroMaxHP = 100
	session.HealingUsed = false
	applyHealing(&session, 40)
	if session.HeroHP != 60 || !session.HealingUsed {
		t.Fatal("codex healing tradeoff missing")
	}
}
func TestFirstBossTrophiesAndConditionalDropsPayOnce(t *testing.T) {
	for _, dungeon := range []string{"crypt", "forge", "void"} {
		data := testRow()
		session := Session{Kind: SessionGate, Dungeon: dungeon, HeroHP: 80, Room: 3, Rooms: 3, EmpoweredFinisher: true}
		first := bonusRelics(&data, &session)
		if len(first) != 2 || first[0].Key != "trophy_"+dungeon {
			t.Fatal("first clean boss does not pay trophy and condition", dungeon, first)
		}
		if len(bonusRelics(&data, &session)) != 0 {
			t.Fatal("boss rewards paid twice", dungeon)
		}
		for _, item := range first {
			if !data.Progress.Discoveries[item.Key] || item.Craftable || atLeast(item.Rank, RankS) {
				t.Fatal("unearned or repeatable top item", item.Key)
			}
		}
	}
}
func TestConditionalLootRequiresActualCombatConditions(t *testing.T) {
	cases := []Session{
		{Kind: SessionGate, Dungeon: "crypt", HeroHP: 80, Room: 3, Rooms: 3, Errors: 1},
		{Kind: SessionGate, Dungeon: "forge", HeroHP: 80, Room: 3, Rooms: 3, HealingUsed: true},
		{Kind: SessionGate, Dungeon: "void", HeroHP: 80, Room: 3, Rooms: 3, EmpoweredFinisher: false},
	}
	for _, session := range cases {
		data := testRow()
		items := bonusRelics(&data, &session)
		if len(items) != 1 || !strings.HasPrefix(items[0].Key, "trophy_") {
			t.Fatal("unmet condition paid", session.Dungeon, items)
		}
	}
	data := testRow()
	session := Session{Kind: SessionGate, Dungeon: "crypt", HeroHP: 80, Room: 3, Rooms: 3, BossHP: 10}
	if len(bonusRelics(&data, &session)) != 0 {
		t.Fatal("living boss paid trophy")
	}
	session.BossHP = 0
	session.HeroHP = 0
	if len(bonusRelics(&data, &session)) != 0 {
		t.Fatal("dead hero paid trophy")
	}
	session.HeroHP = 80
	session.Event = &DungeonEvent{ID: "false_gate", Accepted: true}
	if len(bonusRelics(&data, &session)) != 0 {
		t.Fatal("anomaly pretended to defeat ordinary boss")
	}
}
func TestJournalHintsOnlyRevealAfterTwoBossWinsOrDiscovery(t *testing.T) {
	data := testRow()
	check := func(revealed bool) {
		t.Helper()
		for _, entry := range journalFor(data.Progress) {
			if entry.Dungeon == "crypt" && entry.Revealed != revealed {
				t.Fatal("wrong journal knowledge", entry)
			}
		}
		for _, item := range catalogFor(data.Progress) {
			if item.Key == "vow_unbroken" && strings.Contains(item.ConditionHint, "без единой ошибки") != revealed {
				t.Fatal("catalog leaked condition before journal", item.ConditionHint)
			}
		}
	}
	check(false)
	data.Progress.BossClears["crypt"] = 1
	check(false)
	data.Progress.BossClears["crypt"] = 2
	check(true)
	data.Progress.BossClears["crypt"] = 0
	data.Progress.Discoveries["vow_unbroken"] = true
	check(true)
}
func TestUpgradeComparisonAndLoreMetadataMatchRealNextLevel(t *testing.T) {
	for _, item := range relicCatalog {
		for level := 1; level < 5; level++ {
			view := scaledRelic(item, level)
			next := scaledRelic(item, level+1)
			if view.Lore == "" || view.NextValue != next.Value || view.NextDescription != next.Description {
				t.Fatal("wrong next upgrade metadata", item.Key, level, view.NextValue, next.Value)
			}
		}
		cap := scaledRelic(item, 5)
		if cap.NextValue != 0 || cap.NextDescription != "" {
			t.Fatal("max level advertises another upgrade")
		}
	}
}
func TestFinalBonusLootNeverPaysOnFailure(t *testing.T) {
	data := testRow()
	session := Session{Kind: SessionGate, Dungeon: "crypt", HeroHP: 0, Room: 3, Rooms: 3, BossHP: 30}
	result := (&Service{rnd: seeded(15)}).finishRun(&data, &session, false, time.Now())
	if len(result.BonusRelics) != 0 || len(data.Progress.Discoveries) != 0 {
		t.Fatal("failed run paid bonus rewards")
	}
}
