package farm

import (
	"fmt"
	"time"

	"github.com/danii1kq/srunal/internal/platform/httpapi"
	"github.com/danii1kq/srunal/internal/platform/secure"
)

type Recipe struct {
	Sources []string `json:"sources"`
	Crop    Species  `json:"crop"`
	Base    string   `json:"base"`
}

var Recipes = []Recipe{
	{[]string{"carrot", "radish"}, Species{"aurora_root", "Радужная морковь", "hybrid", 100, 0}, "carrot"},
	{[]string{"tomato", "cucumber"}, Species{"jade_tomato", "Нефритовый томат", "hybrid", 150, 0}, "tomato"},
	{[]string{"strawberry", "blueberry"}, Species{"moonberry", "Лунная клубника", "hybrid", 330, 0}, "strawberry"},
	{[]string{"pumpkin", "watermelon"}, Species{"ghost_pumpkin", "Призрачная тыква", "hybrid", 450, 0}, "pumpkin"},
	{[]string{"corn", "sunflower"}, Species{"solar_corn", "Солнечная кукуруза", "hybrid", 350, 0}, "corn"},
	{[]string{"dragonfruit", "moonflower"}, Species{"crystal_bloom", "Кристальный цветок", "hybrid", 1800, 0}, "moonflower"},
}

type Breeding struct {
	Source    string    `json:"source"`
	Result    string    `json:"result"`
	StartedAt time.Time `json:"started_at"`
	ReadyAt   time.Time `json:"ready_at"`
}
type Trophy struct {
	Plant   Harvest   `json:"plant"`
	SavedAt time.Time `json:"saved_at"`
}

func recipeFor(source string) (Recipe, bool) {
	for _, recipe := range Recipes {
		for _, id := range recipe.Sources {
			if source == id {
				return recipe, true
			}
		}
	}
	return Recipe{}, false
}
func containsID(ids []string, id string) bool {
	for _, value := range ids {
		if value == id {
			return true
		}
	}
	return false
}

func collectionAction(state *State, action Action, now time.Time) (string, error) {
	bad := func(message string) (string, error) { return "", httpapi.BadRequest("farm_collection", message) }
	switch action.Kind {
	case "breed":
		if state.Breeding != nil {
			return bad("Лаборатория уже занята")
		}
		recipe, ok := recipeFor(action.Seed)
		if !ok {
			return bad("Для этого семени пока нет рецепта")
		}
		if state.Seeds[action.Seed] < 2 {
			return bad("Нужно два одинаковых семени")
		}
		state.Seeds[action.Seed] -= 2
		state.Breeding = &Breeding{action.Seed, recipe.Crop.ID, now, now.Add(GrowTime)}
		return "Скрещивание началось. Гибрид будет готов через 3 часа", nil
	case "claim_breed":
		if state.Breeding == nil {
			return bad("Лаборатория пока пуста")
		}
		if state.Breeding.ReadyAt.After(now) {
			return bad("Гибрид ещё не готов")
		}
		id := state.Breeding.Result
		state.Seeds[id]++
		if !containsID(state.Hybrids, id) {
			state.Hybrids = append(state.Hybrids, id)
		}
		state.Breeding = nil
		crop, _ := species(id)
		return "Получено семя: " + crop.Name, nil
	case "keep":
		if len(state.Showcase) >= 6 {
			return bad("На витрине шесть мест. Верни один трофей в амбар")
		}
		for i, item := range state.Barn {
			if item.ID != action.Item {
				continue
			}
			if item.Quantity < 1 {
				return bad("Растение недоступно")
			}
			id, err := secure.NewToken(16)
			if err != nil {
				return "", err
			}
			plant := item
			plant.ID = id
			plant.Quantity = 1
			plant.Value = item.Value / item.Quantity
			state.Showcase = append(state.Showcase, Trophy{plant, now})
			state.Barn[i].Quantity--
			state.Barn[i].Value -= plant.Value
			if state.Barn[i].Quantity == 0 {
				state.Barn = append(state.Barn[:i], state.Barn[i+1:]...)
			}
			return "Экземпляр сохранён на витрине", nil
		}
		return bad("Растение уже перемещено или продано")
	case "return_trophy":
		for i, trophy := range state.Showcase {
			if trophy.Plant.ID != action.Item {
				continue
			}
			state.Barn = append(state.Barn, trophy.Plant)
			state.Showcase = append(state.Showcase[:i], state.Showcase[i+1:]...)
			return "Трофей возвращён в амбар", nil
		}
		return bad("Трофей не найден")
	}
	return bad("Неизвестное действие")
}

func normalizeCollection(state *State) {
	if state.Hybrids == nil {
		state.Hybrids = []string{}
	}
	if state.Showcase == nil {
		state.Showcase = []Trophy{}
	}
	for i := range state.Barn {
		if state.Barn[i].ID == "" {
			state.Barn[i].ID = fmt.Sprintf("legacy-%d", i)
		}
	}
}

func plantWeight(seed string, roll int) int {
	base := 200
	switch seed {
	case "carrot", "radish", "aurora_root":
		base = 180
	case "pumpkin", "ghost_pumpkin":
		base = 4500
	case "watermelon":
		base = 6000
	case "corn", "solar_corn":
		base = 350
	case "strawberry", "moonberry", "blueberry":
		base = 35
	case "dragonfruit":
		base = 550
	}
	return base * (70 + roll) / 100
}
