package farm

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"github.com/danii1kq/srunal/internal/platform/httpapi"
)

const GrowTime = 3 * time.Hour

type Species struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Rarity    string `json:"rarity"`
	Price     int    `json:"price"`
	SeedPrice int    `json:"seed_price"`
}

var Crops = []Species{
	{"carrot", "Морковь", "common", 32, 12},
	{"tomato", "Помидор", "common", 48, 20},
	{"strawberry", "Клубника", "uncommon", 85, 40},
	{"pumpkin", "Тыква", "uncommon", 120, 60},
	{"blueberry", "Голубика", "rare", 210, 110},
	{"dragonfruit", "Драконий фрукт", "legendary", 450, 250},
	{"radish", "Редис", "common", 38, 15},
	{"cucumber", "Огурец", "common", 55, 24},
	{"corn", "Кукуруза", "uncommon", 110, 55},
	{"sunflower", "Подсолнух", "uncommon", 140, 70},
	{"watermelon", "Арбуз", "rare", 300, 150},
	{"moonflower", "Лунный цветок", "legendary", 800, 450},
}

type Plot struct {
	Seed      string    `json:"seed"`
	PlantedAt time.Time `json:"planted_at"`
	ReadyAt   time.Time `json:"ready_at"`
	Care      string    `json:"care"`
	Water     int       `json:"water"`
	Soil      int       `json:"soil"`
	Lamp      int       `json:"lamp"`
}

type Harvest struct {
	Seed     string `json:"seed"`
	Quality  string `json:"quality"`
	Mutation string `json:"mutation"`
	Quantity int    `json:"quantity"`
	Value    int    `json:"value"`
}

type Discovery struct {
	Seed     string `json:"seed"`
	Quality  string `json:"quality"`
	Mutation string `json:"mutation"`
}

type State struct {
	Coins      int                  `json:"coins"`
	XP         int                  `json:"xp"`
	Seeds      map[string]int       `json:"seeds"`
	Plots      []*Plot              `json:"plots"`
	Barn       []Harvest            `json:"barn"`
	Collection map[string]Discovery `json:"collection"`
	Upgrades   map[string]int       `json:"upgrades"`
	Pity       map[string]int       `json:"pity"`
	Opened     int                  `json:"opened"`
	Sold       int                  `json:"sold"`
}

type Action struct {
	Kind    string `json:"kind"`
	Slot    int    `json:"slot"`
	Seed    string `json:"seed"`
	Care    string `json:"care"`
	Pack    string `json:"pack"`
	Upgrade string `json:"upgrade"`
}

type Pack struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Price   int    `json:"price"`
	Weights []int  `json:"weights"`
}

var Packs = []Pack{
	{"garden", "Садовый набор", 75, []int{20, 18, 12, 8, 5, 1, 12, 9, 6, 5, 3, 1}},
	{"exotic", "Экзотический набор", 240, []int{3, 5, 10, 12, 16, 5, 4, 6, 12, 10, 12, 5}},
}

func initialState(now time.Time) State {
	return State{
		Coins: 250,
		Seeds: map[string]int{"carrot": 4, "tomato": 2, "strawberry": 1},
		Plots: []*Plot{
			{Seed: "carrot", PlantedAt: now.Add(-GrowTime), ReadyAt: now, Care: "water"},
			{Seed: "tomato", PlantedAt: now.Add(-GrowTime), ReadyAt: now, Care: "water"},
			nil, nil,
		},
		Barn: []Harvest{}, Collection: map[string]Discovery{},
		Upgrades: map[string]int{"water": 0, "soil": 0, "lamp": 0},
		Pity:     map[string]int{"garden": 0, "exotic": 0},
	}
}

func species(id string) (Species, bool) {
	for _, crop := range Crops {
		if crop.ID == id {
			return crop, true
		}
	}
	return Species{}, false
}

func random(n int) (int, error) {
	value, err := rand.Int(rand.Reader, big.NewInt(int64(n)))
	if err != nil {
		return 0, err
	}
	return int(value.Int64()), nil
}

func upgradeCost(level int) int { return 100 * (level + 1) * (level + 1) }

func outcome(plot Plot, qualityRoll, mutationRoll int) Harvest {
	crop, _ := species(plot.Seed)
	quality, multiplier := "normal", 1
	ideal := 5 + plot.Water*2
	selectChance := 20 + plot.Water*3
	if plot.Care == "water" {
		ideal += 5
		selectChance += 10
	}
	if qualityRoll < ideal {
		quality, multiplier = "perfect", 3
	} else if qualityRoll < ideal+selectChance {
		quality, multiplier = "choice", 2
	}
	mutation, bonus := "none", 1
	chance := 3 + plot.Lamp*2
	if plot.Care == "lamp" {
		chance += 5
	}
	if mutationRoll < chance {
		mutation, bonus = "giant", 2
		if mutationRoll < 1+plot.Lamp {
			mutation, bonus = "gold", 5
		}
		if mutationRoll == 0 {
			mutation, bonus = "crystal", 8
		}
	}
	quantity := 1 + plot.Soil
	if plot.Care == "soil" {
		quantity++
	}
	return Harvest{plot.Seed, quality, mutation, quantity, crop.Price * multiplier * bonus * quantity}
}

// apply mutates only server-owned state while the player row is locked.
func apply(state *State, action Action, now time.Time, roll func(int) (int, error)) (string, []string, int, error) {
	bad := func(message string) (string, []string, int, error) {
		return "", nil, 0, httpapi.BadRequest("farm_action", message)
	}
	spend := func(amount int) bool {
		if state.Coins < amount {
			return false
		}
		state.Coins -= amount
		return true
	}
	switch action.Kind {
	case "plant":
		if action.Slot < 0 || action.Slot >= len(state.Plots) {
			return bad("Грядка не найдена")
		}
		if state.Plots[action.Slot] != nil {
			return bad("Грядка уже занята")
		}
		if _, ok := species(action.Seed); !ok || state.Seeds[action.Seed] <= 0 {
			return bad("Нет таких семян")
		}
		cost := 0
		switch action.Care {
		case "water":
		case "soil":
			cost = 12
		case "lamp":
			cost = 20
		default:
			return bad("Выберите уход")
		}
		if !spend(cost) {
			return bad("Недостаточно монет на уход")
		}
		state.Seeds[action.Seed]--
		state.Plots[action.Slot] = &Plot{action.Seed, now, now.Add(GrowTime), action.Care, state.Upgrades["water"], state.Upgrades["soil"], state.Upgrades["lamp"]}
		return "Семя посажено. Урожай через 3 часа", nil, 0, nil
	case "harvest":
		count := 0
		for i, plot := range state.Plots {
			if plot == nil || plot.ReadyAt.After(now) {
				continue
			}
			q, err := roll(100)
			if err != nil {
				return "", nil, 0, err
			}
			m, err := roll(100)
			if err != nil {
				return "", nil, 0, err
			}
			crop := outcome(*plot, q, m)
			state.Barn = append(state.Barn, crop)
			key := crop.Seed + ":" + crop.Mutation
			state.Collection[key] = Discovery{crop.Seed, crop.Quality, crop.Mutation}
			state.Plots[i] = nil
			state.XP += 10
			count++
		}
		if count == 0 {
			return bad("Урожай ещё не созрел")
		}
		return fmt.Sprintf("Собрано грядок: %d. Урожай ждёт в амбаре", count), nil, 0, nil
	case "sell":
		amount := 0
		for _, item := range state.Barn {
			amount += item.Value
		}
		if amount == 0 {
			return bad("Амбар пока пуст")
		}
		state.Coins += amount
		state.Sold += amount
		state.Barn = []Harvest{}
		return fmt.Sprintf("Урожай продан за %d монет", amount), nil, amount, nil
	case "buy":
		crop, ok := species(action.Seed)
		if !ok {
			return bad("Семена не найдены")
		}
		if !spend(crop.SeedPrice) {
			return bad("Недостаточно монет")
		}
		state.Seeds[crop.ID]++
		return "Семя добавлено в запас", nil, 0, nil
	case "pack":
		var pack *Pack
		for i := range Packs {
			if Packs[i].ID == action.Pack {
				pack = &Packs[i]
			}
		}
		if pack == nil {
			return bad("Набор не найден")
		}
		if !spend(pack.Price) {
			return bad("Недостаточно монет")
		}
		drops := make([]string, 0, 3)
		for i := 0; i < 3; i++ {
			r, err := roll(100)
			if err != nil {
				return "", nil, 0, err
			}
			index, total := 0, 0
			for j, weight := range pack.Weights {
				total += weight
				if r < total {
					index = j
					break
				}
			}
			// Every tenth seed of this pack is rare or legendary if the streak had none.
			isRare := func(index int) bool { return Crops[index].Rarity == "rare" || Crops[index].Rarity == "legendary" }
			if state.Pity[pack.ID] >= 9 && !isRare(index) {
				index = 4
			}
			if isRare(index) {
				state.Pity[pack.ID] = 0
			} else {
				state.Pity[pack.ID]++
			}
			id := Crops[index].ID
			state.Seeds[id]++
			drops = append(drops, id)
		}
		state.Opened++
		return "Три новых семени для твоей фермы", drops, 0, nil
	case "upgrade":
		level, ok := state.Upgrades[action.Upgrade]
		if !ok || level >= 5 {
			return bad("Улучшение недоступно")
		}
		if !spend(upgradeCost(level)) {
			return bad("Недостаточно монет")
		}
		state.Upgrades[action.Upgrade]++
		return "Улучшение действует со следующей посадки", nil, 0, nil
	default:
		return bad("Неизвестное действие")
	}
}
