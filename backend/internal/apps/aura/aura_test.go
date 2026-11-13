package aura

import (
	"fmt"
	"math/rand"
	"testing"
)

func seeded(seed int64) *rand.Rand {
	return rand.New(rand.NewSource(seed))
}

func TestGenerateAlwaysHasFourDistinctOptions(t *testing.T) {
	rnd := seeded(7)
	for level := 1; level <= 60; level++ {
		for i := 0; i < 40; i++ {
			question := Generate(rnd, level, "")
			if len(question.Options) != 4 {
				t.Fatalf("level %d: %d options", level, len(question.Options))
			}
			seen := map[int]bool{}
			found := false
			for _, option := range question.Options {
				if seen[option] {
					t.Fatalf("level %d: duplicate option %d", level, option)
				}
				seen[option] = true
				if option == question.Answer {
					found = true
				}
			}
			if !found {
				t.Fatalf("level %d: answer %d missing from options", level, question.Answer)
			}
			if question.Text == "" || question.TopicName == "" {
				t.Fatalf("level %d: empty text or topic", level)
			}
		}
	}
}

func TestDifficultyBandsMatchTheLevel(t *testing.T) {
	rnd := seeded(11)
	// A beginner must never see algebra or roots.
	for i := 0; i < 200; i++ {
		question := Generate(rnd, 1, "")
		if question.Topic != TopicAdd && question.Topic != TopicSub {
			t.Fatalf("level 1 produced %q", question.Topic)
		}
	}
	// A veteran must never see bare addition.
	for i := 0; i < 200; i++ {
		question := Generate(rnd, 40, "")
		if question.Topic == TopicAdd || question.Topic == TopicSub {
			t.Fatalf("level 40 produced %q", question.Topic)
		}
	}
}

func TestAnswersAreActuallyCorrect(t *testing.T) {
	rnd := seeded(3)
	// Spot-check the closed forms: the generator must not drift from arithmetic.
	for i := 0; i < 300; i++ {
		question := Generate(rnd, 20, TopicSquare)
		// "a²" with answer a*a: verify by parsing the base.
		var base int
		if _, err := fmt.Sscanf(question.Text, "%d²", &base); err != nil {
			t.Fatalf("cannot parse %q: %v", question.Text, err)
		}
		if question.Answer != base*base {
			t.Fatalf("%s answered %d", question.Text, question.Answer)
		}
	}
}

func TestRankRollRespectsTheFloor(t *testing.T) {
	rnd := seeded(5)
	for i := 0; i < 2000; i++ {
		if rank := rollRank(rnd, RankA); !atLeast(rank, RankA) {
			t.Fatalf("floor A produced %q", rank)
		}
		if rank := rollRank(rnd, RankSSS); rank != RankSSS {
			t.Fatalf("floor SSS produced %q", rank)
		}
	}
}

func TestRankRollIsWeightedNotUniform(t *testing.T) {
	rnd := seeded(9)
	counts := map[Rank]int{}
	const rolls = 20000
	for i := 0; i < rolls; i++ {
		counts[rollRank(rnd, RankE)]++
	}
	// E is the most common and SSS the rarest. A uniform roll would break this.
	if counts[RankE] <= counts[RankC] {
		t.Fatalf("E (%d) should beat C (%d)", counts[RankE], counts[RankC])
	}
	if counts[RankC] <= counts[RankS] {
		t.Fatalf("C (%d) should beat S (%d)", counts[RankC], counts[RankS])
	}
	if counts[RankSSS] >= counts[RankSS] {
		t.Fatalf("SSS (%d) should be rarer than SS (%d)", counts[RankSSS], counts[RankSS])
	}
}

func TestEveryRankHasAtLeastOneRelic(t *testing.T) {
	for _, rank := range rankOrder {
		found := false
		for _, relic := range relicCatalog {
			if relic.Rank == rank {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("rank %q has no relic", rank)
		}
	}
}

func TestRelicCatalogKeysAreUnique(t *testing.T) {
	seen := map[string]bool{}
	for _, relic := range relicCatalog {
		if seen[relic.Key] {
			t.Fatalf("duplicate relic key %q", relic.Key)
		}
		seen[relic.Key] = true
		if relic.Value <= 0 {
			t.Fatalf("relic %q has no value", relic.Key)
		}
	}
}

func TestGrindMultiplierDecays(t *testing.T) {
	cases := []struct {
		today int
		want  float64
	}{
		{0, 1}, {14, 1}, {15, 0.5}, {29, 0.5}, {30, 0.2}, {49, 0.2}, {50, 0}, {100, 0},
	}
	for _, item := range cases {
		if got := grindMultiplier(item.today); got != item.want {
			t.Fatalf("today=%d: got %v want %v", item.today, got, item.want)
		}
	}
}

func TestLevelCurveIsMonotonic(t *testing.T) {
	previous := 1
	for xp := 0; xp < 20000; xp += 13 {
		level := levelForXP(xp)
		if level < previous {
			t.Fatalf("level went backwards at xp=%d", xp)
		}
		previous = level
	}
	if levelForXP(0) != 1 {
		t.Fatal("a new hero must start at level 1")
	}
}

func TestSecondsRespectTheClassAndTheFloor(t *testing.T) {
	// A berserker loses three seconds but never drops below the floor.
	berserker := secondsForQuestion(1, ClassBerserker)
	if berserker < 12 {
		t.Fatalf("berserker got %d seconds", berserker)
	}
	strategist := secondsForQuestion(1, ClassStrategist)
	if strategist <= berserker {
		t.Fatalf("strategist (%d) should get more time than berserker (%d)", strategist, berserker)
	}
}

func TestSessionRoundTripsThroughJSON(t *testing.T) {
	session := &Session{
		Kind: SessionArena, Round: 2, Rounds: 7, Combo: 3,
		Question: Question{Text: "2 + 2", Answer: 4, Options: []int{4, 5, 6, 7}, Topic: TopicAdd},
	}
	raw, err := sessionJSON(session)
	if err != nil {
		t.Fatal(err)
	}
	back, err := sessionFromJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if back == nil || back.Kind != session.Kind || back.Round != session.Round {
		t.Fatalf("round trip lost data: %+v", back)
	}
	if back.Question.Answer != 4 {
		t.Fatalf("answer lost: %+v", back.Question)
	}
}

func TestEmptySessionIsNil(t *testing.T) {
	back, err := sessionFromJSON([]byte("{}"))
	if err != nil {
		t.Fatal(err)
	}
	if back != nil {
		t.Fatal("an empty object must mean no session")
	}
}

func TestCritChanceIsCapped(t *testing.T) {
	stats := Stats{Int: 100}
	relics := []Relic{{Passive: "crit", Value: 0.5}}
	if chance := critChance(stats, relics); chance > 0.75 {
		t.Fatalf("crit chance %v exceeds the cap", chance)
	}
}

func TestRelicBonusSumsOnlyTheMatchingPassive(t *testing.T) {
	relics := []Relic{
		{Passive: "farm", Value: 0.1},
		{Passive: "farm", Value: 0.05},
		{Passive: "damage", Value: 0.3},
	}
	if got := relicBonus(relics, "farm"); got != 0.15000000000000002 && got != 0.15 {
		t.Fatalf("farm bonus %v", got)
	}
	if got := relicBonus(relics, "negate"); got != 0 {
		t.Fatalf("negate bonus should be zero, got %v", got)
	}
}
