package farm

type Mutation struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Multiplier  int    `json:"multiplier"`
	MinGenetics int    `json:"min_genetics"`
	Weight      int    `json:"weight"`
	Unlocked    bool   `json:"unlocked"`
}

var MutationTypes = []Mutation{
	{ID: "crystal", Name: "Кристальное", Multiplier: 8, Weight: 10},
	{ID: "gold", Name: "Золотое", Multiplier: 5, Weight: 20},
	{ID: "dew", Name: "Росистое", Multiplier: 2, Weight: 25},
	{ID: "giant", Name: "Гигантское", Multiplier: 2, Weight: 45},
	{ID: "electric", Name: "Грозовое", Multiplier: 6, MinGenetics: 1, Weight: 12},
	{ID: "ancient", Name: "Древнее", Multiplier: 10, MinGenetics: 2, Weight: 8},
	{ID: "rainbow", Name: "Радужное", Multiplier: 12, MinGenetics: 3, Weight: 4},
}

func mutationCatalog(level int) []Mutation {
	result := append([]Mutation(nil), MutationTypes...)
	for i := range result {
		result[i].Unlocked = level >= result[i].MinGenetics
	}
	return result
}

func mutationChance(plot Plot) int {
	chance := 3 + plot.Lamp*2 + plot.Genetics*2
	if plot.Care == "lamp" {
		chance += 5
	}
	return chance
}

func mutationChanceBP(plot Plot) int {
	chance := mutationBP(plot.Lamp, plot.Genetics)
	if plot.Care == "lamp" {
		chance += 500
	}
	return min(chance, 4000)
}

func chooseMutation(level, roll int) (string, int) {
	total := 0
	for _, kind := range MutationTypes {
		if level >= kind.MinGenetics {
			total += kind.Weight
		}
	}
	position := roll * total / 100
	for _, kind := range MutationTypes {
		if level < kind.MinGenetics {
			continue
		}
		if position < kind.Weight {
			return kind.ID, kind.Multiplier
		}
		position -= kind.Weight
	}
	return "giant", 2
}

func showcaseCapacity(state State) int { return 6 + state.Upgrades["gallery"]*2 }
