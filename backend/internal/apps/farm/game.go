package farm

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"github.com/danii1kq/srunal/internal/platform/httpapi"
	"github.com/danii1kq/srunal/internal/platform/secure"
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
	{"pepper", "Сладкий перец", "common", 70, 28},
	{"pear", "Груша", "uncommon", 180, 90},
	{"lavender", "Лаванда", "rare", 380, 210},
	{"lotus", "Лотос", "rare", 620, 350},
	{"mango", "Манго", "legendary", 950, 540},
	{"starfruit", "Карамбола", "legendary", 1400, 800},
}

type Plot struct {
	// A missing yield keeps the original rules for crops already planted before the balance update.
	Yield     int       `json:"yield,omitempty"`
	Genetics  int       `json:"genetics"`
	Nursery   int       `json:"nursery"`
	Seed      string    `json:"seed"`
	PlantedAt time.Time `json:"planted_at"`
	ReadyAt   time.Time `json:"ready_at"`
	Care      string    `json:"care"`
	Water     int       `json:"water"`
	Soil      int       `json:"soil"`
	Lamp      int       `json:"lamp"`
}

type Harvest struct {
	ID       string `json:"id"`
	Weight   int    `json:"weight"`
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
	Crash         *CrashRound          `json:"crash,omitempty"`
	GameHistory   []GameResult         `json:"game_history"`
	GameRequests  []string             `json:"game_requests,omitempty"`
	Expedition    *Expedition          `json:"expedition"`
	Breeding      *Breeding            `json:"breeding"`
	Hybrids       []string             `json:"hybrids"`
	Showcase      []Trophy             `json:"showcase"`
	Theme         string               `json:"theme"`
	OwnedThemes   []string             `json:"owned_themes"`
	Coins         int                  `json:"coins"`
	XP            int                  `json:"xp"`
	Seeds         map[string]int       `json:"seeds"`
	Plots         []*Plot              `json:"plots"`
	Barn          []Harvest            `json:"barn"`
	Collection    map[string]Discovery `json:"collection"`
	Upgrades      map[string]int       `json:"upgrades"`
	Pity          map[string]int       `json:"pity"`
	Opened        int                  `json:"opened"`
	Sold          int                  `json:"sold"`
	ClaimedLevels []int                `json:"claimed_levels"`
	Orders        OrderBoard           `json:"orders"`
	Daily         DailyStreak          `json:"daily"`
}

type Action struct {
	Stake     int    `json:"stake"`
	Side      string `json:"side"`
	RequestID string `json:"request_id"`
	Item      string `json:"item"`
	Theme     string `json:"theme"`
	Code      string `json:"code"`
	Kind      string `json:"kind"`
	Slot      int    `json:"slot"`
	Level     int    `json:"level"`
	Seed      string `json:"seed"`
	Care      string `json:"care"`
	Pack      string `json:"pack"`
	Upgrade   string `json:"upgrade"`
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
	{"garden", "Садовый набор", 75, []int{20, 18, 12, 8, 5, 1, 12, 9, 6, 5, 3, 1, 0, 0, 0, 0, 0, 0}, 10, "rare"},
	{"exotic", "Экзотический набор", 240, []int{3, 5, 10, 12, 16, 5, 4, 6, 12, 10, 12, 5, 0, 0, 0, 0, 0, 0}, 10, "rare"},
	{"lunar", "Лунный набор", 900, []int{0, 0, 5, 5, 25, 15, 0, 0, 5, 5, 25, 15, 0, 0, 0, 0, 0, 0}, 6, "legendary"},
	{"orchard", "Сад чудес", 1100, []int{0, 0, 0, 0, 5, 3, 0, 0, 0, 0, 5, 2, 15, 20, 18, 12, 12, 8}, 10, "rare"},
}

type Upgrade struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Icon          string `json:"icon"`
	Description   string `json:"description"`
	Benefit       string `json:"benefit"`
	MaxLevel      int    `json:"max_level"`
	BaseCost      int    `json:"-"`
	Level         int    `json:"level"`
	Cost          int    `json:"cost"`
	RequiredLevel int    `json:"required_level"`
}

var Upgrades = []Upgrade{
	{ID: "water", Name: "Бережный полив", Icon: "water", Description: "Больше отборных и идеальных растений. После 5-го уровня бонусы растут плавнее.", MaxLevel: 20, BaseCost: 100},
	{ID: "soil", Name: "Плодородная почва", Icon: "soil", Description: "Первые 5 уровней дают по 15% урожайности, следующие по 5%. Каждые 100% гарантируют дополнительное растение, остаток даёт шанс ещё на одно.", MaxLevel: 20, BaseCost: 100},
	{ID: "lamp", Name: "Лаборатория света", Icon: "lamp", Description: "Первые 5 уровней дают по 2 п.п. мутации, следующие по 0,5 п.п.", MaxLevel: 20, BaseCost: 100},
	{ID: "land", Name: "Новые горизонты", Icon: "garden", Description: "Больше места для твоих экспериментов.", Benefit: "+1 грядка за уровень, до 8 грядок", MaxLevel: 4, BaseCost: 300},
	{ID: "market", Name: "Фермерская лавка", Icon: "coin", Description: "Первые 5 уровней дают по 5% к продаже, следующие по 1%.", MaxLevel: 20, BaseCost: 200},
	{ID: "selection", Name: "Секреты селекции", Icon: "spark", Description: "Каждый уровень добавляет 1 п.п. легендарных семян во всех наборах.", MaxLevel: 10, BaseCost: 250},
	{ID: "genetics", Name: "Генетическая станция", Icon: "spark", Description: "Новые мутации на уровнях 1, 2, 3. Первые 5 уровней дают по 2 п.п. мутации, следующие по 1 п.п.", MaxLevel: 10, BaseCost: 350},
	{ID: "nursery", Name: "Заботливый питомник", Icon: "sprout", Description: "Первые 5 уровней дают по 10% возврата семени, следующие по 1%. Возвращается максимум одно семя с грядки.", MaxLevel: 20, BaseCost: 180},
	{ID: "gallery", Name: "Трофейный павильон", Icon: "trophy", Description: "По 2 новых места на витрине за уровень, до 24 мест.", MaxLevel: 9, BaseCost: 300},
}

func upgradesFor(state State) []Upgrade {
	result := append([]Upgrade(nil), Upgrades...)
	for i := range result {
		result[i].Level = state.Upgrades[result[i].ID]
		next := min(result[i].Level+1, result[i].MaxLevel)
		result[i].Cost = upgradeCost(result[i], next)
		result[i].RequiredLevel = upgradeRequiredLevel(next)
		result[i].Benefit = upgradeBenefit(result[i].ID, result[i].Level)
		if result[i].Level < result[i].MaxLevel {
			result[i].Benefit += " → " + upgradeBenefit(result[i].ID, next)
		} else {
			result[i].Cost = 0
		}
	}
	return result
}

func normalizeState(state *State) {
	normalizeCollection(state)
	if state.ClaimedLevels == nil {
		state.ClaimedLevels = []int{}
	}
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
				if crop.Rarity == "legendary" && Packs[i].Weights[j] > 0 {
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
	for _, recipe := range Recipes {
		if recipe.Crop.ID == id {
			return recipe.Crop, true
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

func outcome(plot Plot, qualityRoll, mutationRoll, varietyRoll int) Harvest {
	crop, _ := species(plot.Seed)
	quality, multiplier := "normal", 1
	ideal := 5 + plot.Water*2
	selectChance := 20 + plot.Water*3
	if plot.Yield > 0 {
		ideal, selectChance = perfectPercent(plot.Water), choicePercent(plot.Water)
	}
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
	threshold := mutationChance(plot)
	if plot.Yield > 0 {
		threshold = mutationChanceBP(plot)
	}
	if mutationRoll < threshold {
		mutation, bonus = chooseMutation(plot.Genetics, varietyRoll)
	}
	quantity := 1 + plot.Soil
	if plot.Care == "soil" {
		quantity++
	}
	if plot.Yield > 0 {
		quantity = plot.Yield
	}
	return Harvest{Seed: plot.Seed, Quality: quality, Mutation: mutation, Quantity: quantity, Value: crop.Price * multiplier * bonus * quantity}
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
	case "crash_start", "crash_cashout", "plant_upgrade":
		message, err := casinoAction(state, action, now, roll)
		return message, nil, 0, err
	case "start_expedition", "claim_expedition":
		message, err := expeditionAction(state, action, now)
		return message, nil, 0, err
	case "claim_level":
		message, err := claimLevel(state, action.Level)
		return message, nil, 0, err
	case "claim_daily":
		message, err := claimDaily(state, now)
		return message, nil, 0, err
	case "fulfill_order":
		before := state.Sold
		message, err := fulfillOrder(state, action.Item)
		return message, nil, state.Sold - before, err
	case "breed", "claim_breed", "keep", "return_trophy":
		message, err := collectionAction(state, action, now)
		return message, nil, 0, err
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
		if state.Coins < cost {
			return bad("Недостаточно монет на уход")
		}
		yieldRoll, err := roll(100)
		if err != nil {
			return "", nil, 0, err
		}
		yieldBonus := soilPercent(state.Upgrades["soil"])
		if action.Care == "soil" {
			yieldBonus += 25
		}
		quantity := 1 + yieldBonus/100
		if yieldRoll < yieldBonus%100 {
			quantity++
		}
		spend(cost)
		state.Seeds[action.Seed]--
		state.Plots[action.Slot] = &Plot{Yield: quantity, Seed: action.Seed, PlantedAt: now, ReadyAt: now.Add(GrowTime), Care: action.Care, Water: state.Upgrades["water"], Soil: state.Upgrades["soil"], Lamp: state.Upgrades["lamp"], Genetics: state.Upgrades["genetics"], Nursery: state.Upgrades["nursery"]}
		return "Семя посажено. Урожай через 3 часа", nil, 0, nil
	case "harvest":
		count := 0
		recovered := 0
		for i, plot := range state.Plots {
			if plot == nil || plot.ReadyAt.After(now) {
				continue
			}
			q, err := roll(100)
			if err != nil {
				return "", nil, 0, err
			}
			mutationScale := 100
			if plot.Yield > 0 {
				mutationScale = 10000
			}
			m, err := roll(mutationScale)
			if err != nil {
				return "", nil, 0, err
			}
			variety, err := roll(100)
			if err != nil {
				return "", nil, 0, err
			}
			crop := outcome(*plot, q, m, variety)
			weight, err := roll(100)
			if err != nil {
				return "", nil, 0, err
			}
			crop.Weight = plantWeight(plot.Seed, weight)
			if crop.Mutation == "giant" {
				crop.Weight *= 3
			}
			crop.ID, err = secure.NewToken(16)
			if err != nil {
				return "", nil, 0, err
			}
			state.Barn = append(state.Barn, crop)
			if plot.Nursery > 0 {
				recovery, err := roll(100)
				if err != nil {
					return "", nil, 0, err
				}
				chance := plot.Nursery * 10
				if plot.Yield > 0 {
					chance = nurseryPercent(plot.Nursery)
				}
				if recovery < chance {
					state.Seeds[plot.Seed]++
					recovered++
				}
			}
			key := crop.Seed + ":" + crop.Mutation
			previous, exists := state.Collection[key]
			qualityRank := map[string]int{"normal": 0, "choice": 1, "perfect": 2}
			if !exists || qualityRank[crop.Quality] > qualityRank[previous.Quality] {
				state.Collection[key] = Discovery{crop.Seed, crop.Quality, crop.Mutation}
			}
			state.Plots[i] = nil
			state.XP += 10
			count++
		}
		if count == 0 {
			return bad("Урожай ещё не созрел")
		}
		message := fmt.Sprintf("Собрано грядок: %d. Урожай ждёт в амбаре", count)
		if recovered > 0 {
			message += fmt.Sprintf(". Питомник вернул семян: %d", recovered)
		}
		return message, nil, 0, nil
	case "sell":
		amount := 0
		for _, item := range state.Barn {
			amount += item.Value
		}
		amount = amount * (100 + marketPercent(state.Upgrades["market"])) / 100
		if amount == 0 {
			return bad("Амбар пока пуст")
		}
		state.Coins += amount
		state.Sold += amount
		state.Barn = []Harvest{}
		return fmt.Sprintf("Урожай продан за %d монет", amount), nil, amount, nil
	case "buy":
		crop, ok := species(action.Seed)
		if !ok || crop.SeedPrice <= 0 {
			return bad("Семена не найдены")
		}
		if required := seedUnlockLevel(crop.ID); levelFor(state.XP) < required {
			return bad(fmt.Sprintf("Покупка этих семян откроется на %d уровне", required))
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
		if required := packLevels()[pack.ID]; levelFor(state.XP) < required {
			return bad(fmt.Sprintf("Набор откроется на %d уровне", required))
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
		if levelFor(state.XP) < selected.RequiredLevel {
			return bad(fmt.Sprintf("Следующий уровень улучшения откроется на %d уровне фермы", selected.RequiredLevel))
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
		if action.Upgrade == "gallery" {
			return "На витрине стало больше места", nil, 0, nil
		}
		return "Улучшение действует со следующей посадки", nil, 0, nil
	default:
		return bad("Неизвестное действие")
	}
}
