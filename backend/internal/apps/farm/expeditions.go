package farm

import (
	"fmt"
	"time"

	"github.com/danii1kq/srunal/internal/platform/httpapi"
)

type ExpeditionRoute struct {
	ID              string         `json:"id"`
	Name            string         `json:"name"`
	Description     string         `json:"description"`
	Price           int            `json:"price"`
	DurationSeconds int            `json:"duration_seconds"`
	Seeds           map[string]int `json:"seeds"`
	XP              int            `json:"xp"`
	RequiredLevel   int            `json:"required_level"`
}

type Expedition struct {
	Route     string         `json:"route"`
	Name      string         `json:"name"`
	StartedAt time.Time      `json:"started_at"`
	ReadyAt   time.Time      `json:"ready_at"`
	Seeds     map[string]int `json:"seeds"`
	XP        int            `json:"xp"`
}

var expeditionRoutes = []ExpeditionRoute{
	{ID: "meadow", Name: "Цветущий луг", Description: "Собери ароматные семена среди полевых цветов.", Price: 90, DurationSeconds: int(GrowTime.Seconds()), Seeds: map[string]int{"pepper": 2, "lavender": 1}, XP: 10, RequiredLevel: 6},
	{ID: "orchard", Name: "Забытый сад", Description: "Найди сочные сорта в тени старых плодовых деревьев.", Price: 360, DurationSeconds: int(GrowTime.Seconds()), Seeds: map[string]int{"pear": 2, "mango": 1}, XP: 20, RequiredLevel: 16},
	{ID: "astral", Name: "Звёздные острова", Description: "Вернись с семенами, которые растут под светом созвездий.", Price: 950, DurationSeconds: int(GrowTime.Seconds()), Seeds: map[string]int{"lotus": 1, "starfruit": 2}, XP: 35, RequiredLevel: 20},
}

func expeditionCatalog() []ExpeditionRoute {
	result := append([]ExpeditionRoute(nil), expeditionRoutes...)
	for i := range result {
		result[i].Seeds = copyExpeditionSeeds(result[i].Seeds)
	}
	return result
}

func copyExpeditionSeeds(seeds map[string]int) map[string]int {
	result := make(map[string]int, len(seeds))
	for id, count := range seeds {
		result[id] = count
	}
	return result
}

func expeditionAction(state *State, action Action, now time.Time) (string, error) {
	bad := func(message string) (string, error) { return "", httpapi.BadRequest("farm_expedition", message) }
	switch action.Kind {
	case "start_expedition":
		if state.Expedition != nil {
			return bad("Экспедиция уже в пути. Сначала забери её находки")
		}
		for _, route := range expeditionRoutes {
			if route.ID != action.Item {
				continue
			}
			if levelFor(state.XP) < route.RequiredLevel {
				return bad(fmt.Sprintf("Маршрут откроется на %d уровне", route.RequiredLevel))
			}
			if state.Coins < route.Price {
				return bad("Недостаточно монет на экспедицию")
			}
			state.Coins -= route.Price
			state.Expedition = &Expedition{
				Route: route.ID, Name: route.Name, StartedAt: now,
				ReadyAt: now.Add(time.Duration(route.DurationSeconds) * time.Second),
				Seeds:   copyExpeditionSeeds(route.Seeds), XP: route.XP,
			}
			return "Экспедиция отправилась: " + route.Name + ". Находки через 3 часа", nil
		}
		return bad("Такого маршрута нет")
	case "claim_expedition":
		if state.Expedition == nil {
			return bad("Сейчас нет экспедиции")
		}
		if state.Expedition.ReadyAt.After(now) {
			return bad("Экспедиция ещё в пути")
		}
		if state.Seeds == nil {
			state.Seeds = map[string]int{}
		}
		count := 0
		for id, quantity := range state.Expedition.Seeds {
			state.Seeds[id] += quantity
			count += quantity
		}
		state.XP += state.Expedition.XP
		message := fmt.Sprintf("%s: получено семян %d, +%d опыта", state.Expedition.Name, count, state.Expedition.XP)
		state.Expedition = nil
		return message, nil
	}
	return bad("Неизвестное действие экспедиции")
}
