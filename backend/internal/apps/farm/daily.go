package farm

import (
	"fmt"
	"time"

	"github.com/danii1kq/srunal/internal/platform/httpapi"
)

// DailyStreak tracks consecutive days a player has shown up. The reward grows
// across a seven day cycle and then repeats, so a long absence costs the streak
// but never a permanent bonus.
type DailyStreak struct {
	LastDay  string `json:"last_day"`
	Streak   int    `json:"streak"`
	Best     int    `json:"best"`
	Claimed  int    `json:"claimed"`
}

// DailyReward is one day of the cycle. Seeds are keyed by species id.
type DailyReward struct {
	Day   int            `json:"day"`
	Coins int            `json:"coins"`
	Seeds map[string]int `json:"seeds"`
	XP    int            `json:"xp"`
}

// dailyCycle is deliberately front-loaded: the first days must feel worth the
// tap, and day seven has to be visibly better than six ordinary days.
var dailyCycle = []DailyReward{
	{Day: 1, Coins: 60, XP: 5},
	{Day: 2, Coins: 90, XP: 5},
	{Day: 3, Coins: 130, Seeds: map[string]int{"carrot": 2}, XP: 10},
	{Day: 4, Coins: 180, XP: 10},
	{Day: 5, Coins: 240, Seeds: map[string]int{"strawberry": 2}, XP: 15},
	{Day: 6, Coins: 320, XP: 15},
	{Day: 7, Coins: 500, Seeds: map[string]int{"blueberry": 1}, XP: 25},
}

// dayKey is the UTC calendar day, matching the order board so a player has one
// daily boundary rather than two.
func dayKey(now time.Time) string {
	return now.UTC().Format("2006-01-02")
}

func yesterday(now time.Time) string {
	return now.UTC().AddDate(0, 0, -1).Format("2006-01-02")
}

// dailyState reports where the player stands today without changing anything.
// A missed day resets the streak to zero, but the reset is only persisted when
// the player claims, so simply opening the app does not wipe progress.
func dailyState(state State, now time.Time) (day int, streak int, claimable bool) {
	today := dayKey(now)
	if state.Daily.LastDay == today {
		return cycleDay(state.Daily.Streak), state.Daily.Streak, false
	}
	streak = state.Daily.Streak
	if state.Daily.LastDay != yesterday(now) {
		streak = 0
	}
	return cycleDay(streak + 1), streak + 1, true
}

// cycleDay maps a streak length onto the seven day reward cycle.
func cycleDay(streak int) int {
	if streak < 1 {
		return 1
	}
	return (streak-1)%len(dailyCycle) + 1
}

func rewardForDay(day int) DailyReward {
	for _, reward := range dailyCycle {
		if reward.Day == day {
			return reward
		}
	}
	return dailyCycle[0]
}

// claimDaily grants today's reward exactly once.
func claimDaily(state *State, now time.Time) (string, error) {
	bad := func(message string) (string, error) { return "", httpapi.BadRequest("farm_daily", message) }
	day, streak, claimable := dailyState(*state, now)
	if !claimable {
		return bad("Сегодняшняя награда уже получена")
	}
	reward := rewardForDay(day)
	state.Daily.LastDay = dayKey(now)
	state.Daily.Streak = streak
	state.Daily.Claimed++
	if streak > state.Daily.Best {
		state.Daily.Best = streak
	}
	state.Coins += reward.Coins
	state.XP += reward.XP
	for seed, count := range reward.Seeds {
		state.Seeds[seed] += count
	}
	message := fmt.Sprintf("День %d: +%d монет", day, reward.Coins)
	if reward.XP > 0 {
		message += fmt.Sprintf(", +%d опыта", reward.XP)
	}
	for _, crop := range Crops {
		if count, ok := reward.Seeds[crop.ID]; ok {
			message += fmt.Sprintf(", %s ×%d", crop.Name, count)
		}
	}
	return message, nil
}

// dailyCatalog is the full cycle for the client, with today's position.
func dailyCatalog(state State, now time.Time) map[string]any {
	day, streak, claimable := dailyState(state, now)
	rewards := make([]DailyReward, 0, len(dailyCycle))
	for _, reward := range dailyCycle {
		entry := reward
		entry.Seeds = map[string]int{}
		for seed, count := range reward.Seeds {
			entry.Seeds[seed] = count
		}
		rewards = append(rewards, entry)
	}
	return map[string]any{
		"day":       day,
		"streak":    streak,
		"best":      state.Daily.Best,
		"claimable": claimable,
		"rewards":   rewards,
	}
}
