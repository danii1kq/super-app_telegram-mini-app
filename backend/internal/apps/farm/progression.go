package farm

import "fmt"

func seedUnlockLevel(id string) int {
	switch id {
	case "strawberry", "pumpkin", "corn", "sunflower", "pear":
		return 3
	case "blueberry", "watermelon", "lavender":
		return 6
	case "dragonfruit":
		return 9
	case "lotus":
		return 10
	case "moonflower":
		return 12
	case "mango":
		return 16
	case "starfruit":
		return 20
	}
	for _, recipe := range Recipes {
		if recipe.Crop.ID == id {
			return max(seedUnlockLevel(recipe.Sources[0]), seedUnlockLevel(recipe.Sources[1])) + 2
		}
	}
	return 1
}

func seedLevels() map[string]int {
	levels := map[string]int{}
	for _, crop := range Crops {
		levels[crop.ID] = seedUnlockLevel(crop.ID)
	}
	for _, recipe := range Recipes {
		levels[recipe.Crop.ID] = seedUnlockLevel(recipe.Crop.ID)
	}
	return levels
}

func packLevels() map[string]int {
	return map[string]int{"garden": 1, "exotic": 6, "lunar": 12, "orchard": 16}
}

func upgradeCost(upgrade Upgrade, next int) int {
	cost := upgrade.BaseCost * next * next
	if next > 5 {
		cost = cost * (next - 3) * (next + 5) / 10
	}
	return cost
}

func upgradeRequiredLevel(next int) int {
	if next <= 5 {
		return 1 + (next-1)*2
	}
	return next * 2
}

func soilPercent(level int) int    { return min(level, 5)*15 + max(level-5, 0)*5 }
func marketPercent(level int) int  { return min(level, 5)*5 + max(level-5, 0) }
func nurseryPercent(level int) int { return min(level, 5)*10 + max(level-5, 0) }
func perfectPercent(level int) int { return 5 + min(level, 5)*2 + max(level-5, 0) }
func choicePercent(level int) int  { return 20 + min(level, 5)*3 + max(level-5, 0) }
func mutationBP(lamp, genetics int) int {
	return 300 + min(lamp, 5)*200 + max(lamp-5, 0)*50 + min(genetics, 5)*200 + max(genetics-5, 0)*100
}

func upgradeBenefit(id string, level int) string {
	switch id {
	case "water":
		return fmt.Sprintf("Идеальное %d%%, отборное %d%%", perfectPercent(level), choicePercent(level))
	case "soil":
		return fmt.Sprintf("Урожайность +%d%%", soilPercent(level))
	case "lamp":
		return fmt.Sprintf("Шанс мутации +%.1f п.п.", float64(mutationBP(level, 0)-300)/100)
	case "land":
		return fmt.Sprintf("Грядки: %d", 4+level)
	case "market":
		return fmt.Sprintf("Цена продажи +%d%%", marketPercent(level))
	case "selection":
		return fmt.Sprintf("Легендарные семена +%d п.п.", level)
	case "genetics":
		return fmt.Sprintf("Мутации +%d п.п.; %d/3 новых вида", (mutationBP(0, level)-300)/100, min(level, 3))
	case "nursery":
		return fmt.Sprintf("Возврат семени %d%%", nurseryPercent(level))
	case "gallery":
		return fmt.Sprintf("Мест на витрине: %d", 6+level*2)
	}
	return ""
}

func farmBonuses(state State) map[string]any {
	u := state.Upgrades
	return map[string]any{
		"perfect_percent": perfectPercent(u["water"]), "choice_percent": choicePercent(u["water"]),
		"yield_percent": soilPercent(u["soil"]), "market_percent": marketPercent(u["market"]),
		"nursery_percent": nurseryPercent(u["nursery"]), "mutation_percent": float64(mutationBP(u["lamp"], u["genetics"])) / 100,
	}
}
