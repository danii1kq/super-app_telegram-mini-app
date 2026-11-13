package farm

import (
	"testing"
	"time"
)

func day(n int) time.Time {
	return time.Date(2026, 11, n, 10, 0, 0, 0, time.UTC)
}

func TestDailyCycleRepeatsEverySevenDays(t *testing.T) {
	for streak := 1; streak <= 21; streak++ {
		got := cycleDay(streak)
		want := (streak-1)%7 + 1
		if got != want {
			t.Fatalf("streak %d mapped to day %d, want %d", streak, got, want)
		}
	}
	if cycleDay(0) != 1 {
		t.Fatal("an empty streak must offer day 1")
	}
}

func TestDailyClaimGrowsStreakAndPaysOnce(t *testing.T) {
	s := initialState(day(10))
	coins, xp := s.Coins, s.XP

	if _, err := claimDaily(&s, day(10)); err != nil {
		t.Fatal(err)
	}
	if s.Daily.Streak != 1 || s.Daily.LastDay != "2026-11-10" {
		t.Fatalf("first claim did not start the streak: %+v", s.Daily)
	}
	if s.Coins != coins+dailyCycle[0].Coins || s.XP != xp+dailyCycle[0].XP {
		t.Fatal("day 1 reward was not paid exactly")
	}
	if _, err := claimDaily(&s, day(10)); err == nil {
		t.Fatal("the same day paid twice")
	}

	// Next day continues the streak.
	if _, err := claimDaily(&s, day(11)); err != nil {
		t.Fatal(err)
	}
	if s.Daily.Streak != 2 {
		t.Fatalf("streak did not continue: %d", s.Daily.Streak)
	}
	if s.Daily.Best != 2 {
		t.Fatalf("best streak not tracked: %d", s.Daily.Best)
	}
}

func TestDailyStreakResetsAfterAMissedDay(t *testing.T) {
	s := initialState(day(10))
	for offset := 0; offset < 3; offset++ {
		if _, err := claimDaily(&s, day(10+offset)); err != nil {
			t.Fatal(err)
		}
	}
	if s.Daily.Streak != 3 {
		t.Fatalf("expected a 3 day streak, got %d", s.Daily.Streak)
	}
	// Skip day 13 entirely and claim on 14.
	before := s.Daily.Best
	if _, err := claimDaily(&s, day(14)); err != nil {
		t.Fatal(err)
	}
	if s.Daily.Streak != 1 {
		t.Fatalf("a missed day must reset the streak, got %d", s.Daily.Streak)
	}
	if s.Daily.Best != before {
		t.Fatal("the best streak must survive a reset")
	}
}

func TestDailyStateDoesNotMutateBeforeClaiming(t *testing.T) {
	s := initialState(day(10))
	if _, err := claimDaily(&s, day(10)); err != nil {
		t.Fatal(err)
	}
	saved := s.Daily
	// Merely opening the app two days later must not wipe the saved streak.
	_, streak, claimable := dailyState(s, day(12))
	if !claimable {
		t.Fatal("a new day must be claimable")
	}
	if streak != 1 {
		t.Fatalf("a fresh streak should be 1 after a gap, got %d", streak)
	}
	if s.Daily != saved {
		t.Fatal("dailyState changed the saved state")
	}
}

func TestDailyRewardCycleIsFrontLoaded(t *testing.T) {
	previous := 0
	for _, reward := range dailyCycle {
		if reward.Coins <= previous {
			t.Fatalf("day %d does not pay more than the day before", reward.Day)
		}
		previous = reward.Coins
	}
	if len(dailyCycle) != 7 {
		t.Fatalf("expected a seven day cycle, got %d", len(dailyCycle))
	}
	last := dailyCycle[len(dailyCycle)-1]
	if len(last.Seeds) == 0 {
		t.Fatal("the seventh day should carry a seed, not only coins")
	}
}

func TestDailyCatalogReportsPosition(t *testing.T) {
	s := initialState(day(10))
	catalog := dailyCatalog(s, day(10))
	if catalog["day"].(int) != 1 || catalog["claimable"].(bool) != true {
		t.Fatalf("unexpected catalog: %+v", catalog)
	}
	rewards := catalog["rewards"].([]DailyReward)
	if len(rewards) != 7 {
		t.Fatalf("catalog must expose the whole cycle, got %d", len(rewards))
	}
	if _, err := claimDaily(&s, day(10)); err != nil {
		t.Fatal(err)
	}
	after := dailyCatalog(s, day(10))
	if after["claimable"].(bool) {
		t.Fatal("a claimed day still reports claimable")
	}
}

func TestDailySeedsLandInTheInventory(t *testing.T) {
	s := initialState(day(10))
	// Walk to day 3, which carries carrot seeds.
	for offset := 0; offset < 3; offset++ {
		if _, err := claimDaily(&s, day(10+offset)); err != nil {
			t.Fatal(err)
		}
	}
	want := initialState(day(10)).Seeds["carrot"] + dailyCycle[2].Seeds["carrot"]
	if s.Seeds["carrot"] != want {
		t.Fatalf("day 3 seeds not granted: have %d want %d", s.Seeds["carrot"], want)
	}
}
