package farm

import (
	"testing"
	"time"
)

func TestLevelCurveIsMonotonicAndCapped(t *testing.T) {
	if levelFor(0) != 1 {
		t.Fatal("a new player must start at level 1")
	}
	previous := 1
	for xp := 0; xp < 5000; xp += 7 {
		level := levelFor(xp)
		if level < previous {
			t.Fatalf("level went backwards at xp=%d", xp)
		}
		if level > MaxLevel {
			t.Fatalf("level %d exceeds the cap at xp=%d", level, xp)
		}
		previous = level
	}
	if levelFor(1_000_000) != MaxLevel {
		t.Fatal("a large xp total must reach the cap")
	}
}

func TestLevelProgressMatchesCurve(t *testing.T) {
	for level := 1; level < MaxLevel; level++ {
		spent := 0
		for current := 1; current < level; current++ {
			spent += levelNeed(current)
		}
		gotLevel, into, need := levelProgress(spent)
		if gotLevel != level || into != 0 || need != levelNeed(level) {
			t.Fatalf("boundary of level %d reported %d/%d/%d", level, gotLevel, into, need)
		}
		_, into, _ = levelProgress(spent + levelNeed(level) - 1)
		if into != levelNeed(level)-1 {
			t.Fatalf("last xp of level %d reported %d", level, into)
		}
	}
	if level, into, need := levelProgress(1_000_000); level != MaxLevel || into != 0 || need != 0 {
		t.Fatalf("capped level must report a finished bar, got %d/%d/%d", level, into, need)
	}
}

func TestTitleFollowsHighestMilestone(t *testing.T) {
	if titleFor(1) != "Новичок" {
		t.Fatal("level 1 title")
	}
	if titleFor(4) != "Садовод" {
		t.Fatal("level 4 must keep the level 3 title")
	}
	if titleFor(MaxLevel) != "Легенда сада" {
		t.Fatal("max level title")
	}
}

func TestClaimLevelGrantsOnceAndOnlyWhenReached(t *testing.T) {
	now := time.Now()
	s := initialState(now)

	if _, err := claimLevel(&s, 2); err == nil {
		t.Fatal("claimed a level that was not reached")
	}

	s.XP = levelNeed(1) + levelNeed(2)
	if levelFor(s.XP) != 3 {
		t.Fatalf("expected level 3, got %d", levelFor(s.XP))
	}
	if pending := claimableLevels(s); len(pending) != 2 || pending[0] != 2 || pending[1] != 3 {
		t.Fatalf("pending levels out of order: %v", pending)
	}

	coins := s.Coins
	carrots := s.Seeds["carrot"]
	if _, err := claimLevel(&s, 2); err != nil {
		t.Fatal(err)
	}
	if s.Coins != coins+150 || s.Seeds["carrot"] != carrots+3 {
		t.Fatal("level 2 reward was not granted exactly")
	}
	if _, err := claimLevel(&s, 2); err == nil {
		t.Fatal("the same level paid twice")
	}
	if pending := claimableLevels(s); len(pending) != 1 || pending[0] != 3 {
		t.Fatalf("level 3 should still be pending, got %v", pending)
	}
}

func TestLevelRewardAddsPlotAndRespectsLandCap(t *testing.T) {
	now := time.Now()
	s := initialState(now)
	s.XP = 100_000
	plots := len(s.Plots)

	if _, err := claimLevel(&s, 5); err != nil {
		t.Fatal(err)
	}
	if len(s.Plots) != plots+1 || s.Upgrades["land"] != 1 {
		t.Fatal("level 5 must add one bed through the land upgrade")
	}

	// Fill the land upgrade to its cap, then claim level 8: the bed cannot be
	// added, so the reward must pay coins instead of vanishing.
	s.Upgrades["land"] = landMaxLevel()
	before := len(s.Plots)
	coins := s.Coins
	if _, err := claimLevel(&s, 8); err != nil {
		t.Fatal(err)
	}
	if len(s.Plots) != before {
		t.Fatal("garden grew past the land cap")
	}
	if s.Coins != coins+1800+landBaseCost() {
		t.Fatalf("capped plot reward was not converted to coins, got %d", s.Coins-coins)
	}
}

func TestClaimLevelRejectsUnknownLevels(t *testing.T) {
	s := initialState(time.Now())
	s.XP = 100_000
	for _, level := range []int{0, 1, MaxLevel + 1, 99} {
		if _, err := claimLevel(&s, level); err == nil {
			t.Fatalf("level %d was accepted", level)
		}
	}
}

func TestLevelCatalogCoversEveryReward(t *testing.T) {
	catalog := levelCatalog(initialState(time.Now()))
	if len(catalog) != len(levelRewards) {
		t.Fatalf("catalog has %d entries, rewards have %d", len(catalog), len(levelRewards))
	}
	for _, entry := range catalog {
		if entry.Level < 2 || entry.Level > MaxLevel {
			t.Fatalf("unexpected level in catalog: %d", entry.Level)
		}
		if entry.Coins <= 0 && len(entry.Seeds) == 0 && entry.Plots == 0 {
			t.Fatalf("level %d gives nothing", entry.Level)
		}
	}
}
