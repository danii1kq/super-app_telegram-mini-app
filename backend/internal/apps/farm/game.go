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
	Theme       string               `json:"theme"`
	OwnedThemes []string             `json:"owned_themes"`
	Coins       int                  `json:"coins"`
	XP          int                  `json:"xp"`
	Seeds       map[string]int       `json:"seeds"`
	Plots       []*Plot              `json:"plots"`
	Barn        []Harvest            `json:"barn"`
	Collection  map[string]Discovery `json:"collection"`
	Upgrades    map[string]int       `json:"upgrades"`
	Pity        map[string]int       `json:"pity"`
	Opened      int                  `json:"opened"`
	Sold        int                  `json:"sold"`
}

type Action struct {
	Theme   string `json:"theme"`
	Code    string `json:"code"`
	Kind    string `json:"kind"`
	Slot    int    `json:"slot"`
	Seed    string `json:"seed"`
	Care    string `json:"care"`
	Pack    string `json:"pack"`
	Upgrade string `json:"upgrade"`
}

type Pack struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Price     int    `json:"price"`
	Weights   []int  `json:"weights"`
	PityAfter int    `json:"pity_after"`
	Guarantee string `json:"guarantee"`
}

var Packs = []Pack{
	{"garden", "Садовый набор", 75, []int{20, 18, 12, 8, 5, 1, 12, 9, 6, 5, 3, 1}, 10, "rare"},
	{"exotic", "Экзотический набор", 240, []int{3, 5, 10, 12, 16, 5, 4, 6, 12, 10, 12, 5}, 10, "rare"},
	{"lunar", "Лунный набор", 900, []int{0, 0, 5, 5, 25, 15, 0, 0, 5, 5, 25, 15}, 6, "legendary"},
}

type Upgrade struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Icon        string `json:"icon"`
	Description string `json:"description"`
	Benefit     string `json:"benefit"`
	MaxLevel    int    `json:"max_level"`
	BaseCost    int    `json:"-"`
	Level       int    `json:"level"`
	Cost        int    `json:"cost"`
}

var Upgrades = []Upgrade{
	{ID: "water", Name: "Бережный полив", Icon: "water", Description: "Больше отборных и идеальных растений.", Benefit: "+2% идеального и +3% отборного за уровень", MaxLevel: 5, BaseCost: 100},
	{ID: "soil", Name: "Плодородная почва", Icon: "soil", Description: "Больше урожая с каждой грядки.", Benefit: "+1 растение с грядки за уровень", MaxLevel: 5, BaseCost: 100},
	{ID: "lamp", Name: "Лаборатория света", Icon: "lamp", Description: "Повышает шанс необычных мутаций.", Benefit: "+2% к шансу мутации за уровень", MaxLevel: 5, BaseCost: 100},
	{ID: "land", Name: "Новые горизонты", Icon: "garden", Description: "Больше места для твоих экспериментов.", Benefit: "+1 грядка за уровень, до 8 грядок", MaxLevel: 4, BaseCost: 300},
	{ID: "market", Name: "Фермерская лавка", Icon: "coin", Description: "Продавай свой урожай дороже.", Benefit: "+5% к цене продажи за уровень", MaxLevel: 5, BaseCost: 200},
	{ID: "selection", Name: "Секреты селекции", Icon: "spark", Description: "Повышает шансы легендарных семян во всех наборах.", Benefit: "+1 п.п. к легендарному шансу за уровень", MaxLevel: 5, BaseCost: 250},
}

func upgradesFor(state State) []Upgrade {
	result := append([]Upgrade(nil), Upgrades...)
	for i := range result {
		result[i].Level = state.Upgrades[result[i].ID]
		result[i].Cost = result[i].BaseCost * (result[i].Level + 1) * (result[i].Level + 1)
	}
	return result
}

func normalizeState(state *State) {
	if state.Theme == "" {
		state.Theme = "garden"
	}
	if !ownsTheme(*state, "garden") {
		state.OwnedThemes = append(state.OwnedThemes, "garden")
	}
	if state.Upgrades == nil {
		state.Upgrades = map[string]int{}
	}
	if state.Pity == nil {
		state.Pity = map[string]int{}
	}
	for _, upgrade := range Upgrades {
		if _, ok := state.Upgrades[upgrade.ID]; !ok {
			state.Upgrades[upgrade.ID] = 0
		}
	}
	for _, pack := range Packs {
		if _, ok := state.Pity[pack.ID]; !ok {
			state.Pity[pack.ID] = 0
		}
	}
}

func packsFor(level int) []Pack {
	result := append([]Pack(nil), Packs...)
	for i := range result {
		result[i].Weights = append([]int(nil), result[i].Weights...)
		for step := 0; step < level; step++ {
			donor := -1
			for j, crop := range Crops {
				if crop.Rarity != "legendary" && result[i].Weights[j] > 0 && (donor < 0 || result[i].Weights[j] > result[i].Weights[donor]) {
					donor = j
				}
			}
			if donor < 0 {
				break
			}
			result[i].Weights[donor]--
			legendary := []int{}
			for j, crop := range Crops {
				if crop.Rarity == "legendary" {
					legendary = append(legendary, j)
				}
			}
			result[i].Weights[legendary[step%len(legendary)]]++
		}
	}
	return result
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
	normalizeState(state)
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
	case "theme":
		if _, ok := findTheme(action.Theme); !ok || !ownsTheme(*state, action.Theme) {
			return bad("Эта тема пока не открыта")
		}
		state.Theme = action.Theme
		return "Тема фермы изменена", nil, 0, nil
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
		amount = amount * (100 + state.Upgrades["market"]*5) / 100
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
		available := packsFor(state.Upgrades["selection"])
		for i := range available {
			if available[i].ID == action.Pack {
				pack = &available[i]
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
			qualifies := func(index int) bool {
				return Crops[index].Rarity == "legendary" || (pack.Guarantee == "rare" && Crops[index].Rarity == "rare")
			}
			if state.Pity[pack.ID] >= pack.PityAfter-1 && !qualifies(index) {
				for j := range Crops {
					if qualifies(j) {
						index = j
						break
					}
				}
			}
			if qualifies(index) {
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
		var selected *Upgrade
		for _, upgrade := range upgradesFor(*state) {
			if upgrade.ID == action.Upgrade {
				value := upgrade
				selected = &value
				break
			}
		}
		if selected == nil || selected.Level >= selected.MaxLevel {
			return bad("Улучшение недоступно")
		}
		if !spend(selected.Cost) {
			return bad("Недостаточно монет")
		}
		state.Upgrades[action.Upgrade]++
		if action.Upgrade == "land" {
			state.Plots = append(state.Plots, nil)
			return "Новая грядка готова к посадке", nil, 0, nil
		}
		if action.Upgrade == "selection" {
			return "Шанс легендарных семян в наборах повышен", nil, 0, nil
		}
		if action.Upgrade == "market" {
			return "Лавка приносит больше монет с каждой продажи", nil, 0, nil
		}
		return "Улучшение действует со следующей посадки", nil, 0, nil
	default:
		return bad("Неизвестное действие")
	}
}
