package farm

import (
	"fmt"

	"github.com/danii1kq/srunal/internal/platform/httpapi"
)

// MaxLevel caps progression. Rewards stop there but XP keeps accumulating so
// the bar never freezes at a full track.
const MaxLevel = 10

// LevelReward is what a player receives once, on reaching Level.
//
// Plots is the number of extra garden beds granted. It raises the land upgrade
// level rather than the plot slice directly, so the existing cap and price
// logic stays the single source of truth for how many beds exist.
type LevelReward struct {
	Level int            `json:"level"`
	Title string         `json:"title"`
	Coins int            `json:"coins"`
	Seeds map[string]int `json:"seeds"`
	Plots int            `json:"plots"`
}

// Titles are cumulative milestones: a player keeps the title of the highest
// level they have reached.
var levelTitles = map[int]string{
	1:  "Новичок",
	3:  "Садовод",
	5:  "Хозяин фермы",
	7:  "Мастер урожая",
	9:  "Легенда сада",
}

var levelRewards = []LevelReward{
	{Level: 2, Coins: 150, Seeds: map[string]int{"carrot": 3}},
	{Level: 3, Coins: 250, Seeds: map[string]int{"strawberry": 2}},
	{Level: 4, Coins: 400, Seeds: map[string]int{"pumpkin": 2}},
	{Level: 5, Coins: 600, Plots: 1},
	{Level: 6, Coins: 900, Seeds: map[string]int{"blueberry": 2}},
	{Level: 7, Coins: 1300, Seeds: map[string]int{"watermelon": 2}},
	{Level: 8, Coins: 1800, Plots: 1},
	{Level: 9, Coins: 2500, Seeds: map[string]int{"dragonfruit": 1}},
	{Level: 10, Coins: 4000, Seeds: map[string]int{"moonflower": 1}},
}

// levelNeed is the XP required to move from level to level+1. The first levels
// are quick so a new player sees the bar move on the first day, then it grows.
func levelNeed(level int) int {
	if level < 1 {
		level = 1
	}
	return 100 + (level-1)*40
}

// levelFor returns the level earned by a total XP amount.
func levelFor(xp int) int {
	if xp < 0 {
		return 1
	}
	level, remaining := 1, xp
	for level < MaxLevel && remaining >= levelNeed(level) {
		remaining -= levelNeed(level)
		level++
	}
	return level
}

// levelProgress reports the position inside the current level for the client.
func levelProgress(xp int) (level, into, need int) {
	level = levelFor(xp)
	if level >= MaxLevel {
		return level, 0, 0
	}
	spent := 0
	for current := 1; current < level; current++ {
		spent += levelNeed(current)
	}
	return level, xp - spent, levelNeed(level)
}

// titleFor returns the milestone title for a level.
func titleFor(level int) string {
	title := levelTitles[1]
	for milestone := 1; milestone <= level; milestone++ {
		if name, ok := levelTitles[milestone]; ok {
			title = name
		}
	}
	return title
}

func rewardFor(level int) (LevelReward, bool) {
	for _, reward := range levelRewards {
		if reward.Level == level {
			return reward, true
		}
	}
	return LevelReward{}, false
}

func claimedLevel(state State, level int) bool {
	for _, done := range state.ClaimedLevels {
		if done == level {
			return true
		}
	}
	return false
}

// claimableLevels lists reached levels whose reward is still unclaimed, oldest
// first, so the client can offer them in order.
func claimableLevels(state State) []int {
	pending := []int{}
	for _, reward := range levelRewards {
		if reward.Level <= levelFor(state.XP) && !claimedLevel(state, reward.Level) {
			pending = append(pending, reward.Level)
		}
	}
	return pending
}

// levelCatalog is the full reward table for the client, with a claimed flag.
func levelCatalog(state State) []LevelReward {
	result := make([]LevelReward, 0, len(levelRewards))
	for _, reward := range levelRewards {
		entry := reward
		entry.Seeds = map[string]int{}
		for seed, count := range reward.Seeds {
			entry.Seeds[seed] = count
		}
		result = append(result, entry)
	}
	return result
}

// claimLevel grants one level reward exactly once.
func claimLevel(state *State, level int) (string, error) {
	bad := func(message string) (string, error) { return "", httpapi.BadRequest("farm_level", message) }
	if level < 2 || level > MaxLevel {
		return bad("Такого уровня нет")
	}
	if levelFor(state.XP) < level {
		return bad("Уровень ещё не достигнут")
	}
	if claimedLevel(*state, level) {
		return bad("Награда за этот уровень уже получена")
	}
	reward, ok := rewardFor(level)
	if !ok {
		return bad("Награда не найдена")
	}
	state.ClaimedLevels = append(state.ClaimedLevels, level)
	state.Coins += reward.Coins
	for seed, count := range reward.Seeds {
		state.Seeds[seed] += count
	}
	granted := 0
	for i := 0; i < reward.Plots; i++ {
		if state.Upgrades["land"] >= landMaxLevel() {
			// The garden is already at its widest; pay the difference instead of
			// silently dropping the reward.
			state.Coins += landBaseCost()
			continue
		}
		state.Upgrades["land"]++
		state.Plots = append(state.Plots, nil)
		granted++
	}
	message := fmt.Sprintf("Уровень %d: %s", level, titleFor(level))
	if reward.Coins > 0 {
		message += fmt.Sprintf(", +%d монет", reward.Coins)
	}
	if len(reward.Seeds) > 0 {
		for _, crop := range Crops {
			if count, ok := reward.Seeds[crop.ID]; ok {
				message += fmt.Sprintf(", %s ×%d", crop.Name, count)
			}
		}
	}
	if granted > 0 {
		message += fmt.Sprintf(", +%d грядка", granted)
	}
	return message, nil
}

// landMaxLevel and landBaseCost mirror the land entry in the upgrade table so
// the reward path cannot drift from the shop.
func landMaxLevel() int {
	for _, upgrade := range Upgrades {
		if upgrade.ID == "land" {
			return upgrade.MaxLevel
		}
	}
	return 0
}

func landBaseCost() int {
	for _, upgrade := range Upgrades {
		if upgrade.ID == "land" {
			return upgrade.BaseCost
		}
	}
	return 0
}
