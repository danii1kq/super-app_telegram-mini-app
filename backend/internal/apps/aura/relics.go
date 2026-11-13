package aura

import (
	"fmt"
	"math/rand"
)

// Rank is the relic tier. The order matters: it drives the roll weights and the
// guarantee that a long dry streak ends.
type Rank string

const (
	RankE   Rank = "E"
	RankD   Rank = "D"
	RankC   Rank = "C"
	RankB   Rank = "B"
	RankA   Rank = "A"
	RankS   Rank = "S"
	RankSS  Rank = "SS"
	RankSSS Rank = "SSS"
)

// rankOrder is ascending, so a comparison is an index lookup.
var rankOrder = []Rank{RankE, RankD, RankC, RankB, RankA, RankS, RankSS, RankSSS}

func rankIndex(rank Rank) int {
	for i, value := range rankOrder {
		if value == rank {
			return i
		}
	}
	return 0
}

// atLeast reports whether rank is at least floor.
func atLeast(rank, floor Rank) bool {
	return rankIndex(rank) >= rankIndex(floor)
}

// Slot is where a relic is worn. Only a worn relic counts.
type Slot string

const (
	SlotWeapon    Slot = "weapon"
	SlotArmor     Slot = "armor"
	SlotAccessory Slot = "accessory"
)

// Relic is one collectible. Passive is the numeric bonus the engine applies.
type Relic struct {
	UpgradeMaterials map[string]int `json:"upgrade_materials"`
	Description      string         `json:"description"`
	Effect           string         `json:"effect,omitempty"`
	UpgradeCost      int            `json:"upgrade_cost"`
	CraftCost        map[string]int `json:"craft_cost"`
	CraftAura        int            `json:"craft_aura"`
	Key              string         `json:"key"`
	Name             string         `json:"name"`
	Rank             Rank           `json:"rank"`
	Slot             Slot           `json:"slot"`
	Passive          string         `json:"passive"`
	Value            float64        `json:"value"`
}

// relicCatalog is the deck. Names are manga-flavoured on purpose: the original
// game leaned on that and it reads better than "Sword +1".
var relicCatalog = []Relic{
	// Weapons: damage and crit.
	{Key: "rusty_blade", Name: "Ржавый клинок", Rank: RankE, Slot: SlotWeapon, Passive: "damage", Value: 0.03},
	{Key: "apprentice_staff", Name: "Посох ученика", Rank: RankE, Slot: SlotWeapon, Passive: "crit", Value: 0.02},
	{Key: "hunters_bow", Name: "Лук охотника", Rank: RankD, Slot: SlotWeapon, Passive: "damage", Value: 0.05},
	{Key: "twin_daggers", Name: "Парные клинки", Rank: RankC, Slot: SlotWeapon, Passive: "crit", Value: 0.06},
	{Key: "storm_spear", Name: "Копьё бури", Rank: RankB, Slot: SlotWeapon, Passive: "damage", Value: 0.10},
	{Key: "moonlit_saber", Name: "Лунная сабля", Rank: RankA, Slot: SlotWeapon, Passive: "crit", Value: 0.12},
	{Key: "void_reaver", Name: "Косарь пустоты", Rank: RankS, Slot: SlotWeapon, Passive: "damage", Value: 0.18},
	{Key: "dragon_fang", Name: "Клык дракона", Rank: RankSS, Slot: SlotWeapon, Passive: "damage", Value: 0.25},
	{Key: "world_ender", Name: "Губитель миров", Rank: RankSSS, Slot: SlotWeapon, Passive: "damage", Value: 0.35},

	// Armor: durability and negation.
	{Key: "cloth_robe", Name: "Тканевая роба", Rank: RankE, Slot: SlotArmor, Passive: "negate", Value: 0.02},
	{Key: "leather_vest", Name: "Кожаный жилет", Rank: RankD, Slot: SlotArmor, Passive: "negate", Value: 0.04},
	{Key: "chain_mail", Name: "Кольчуга", Rank: RankC, Slot: SlotArmor, Passive: "negate", Value: 0.06},
	{Key: "guardian_plate", Name: "Латы стража", Rank: RankB, Slot: SlotArmor, Passive: "negate", Value: 0.09},
	{Key: "frost_aegis", Name: "Морозная эгида", Rank: RankA, Slot: SlotArmor, Passive: "negate", Value: 0.12},
	{Key: "astral_shroud", Name: "Астральный саван", Rank: RankS, Slot: SlotArmor, Passive: "negate", Value: 0.16},
	{Key: "eternal_bulwark", Name: "Вечный оплот", Rank: RankSS, Slot: SlotArmor, Passive: "negate", Value: 0.22},
	{Key: "world_shell", Name: "Панцирь мира", Rank: RankSSS, Slot: SlotArmor, Passive: "negate", Value: 0.30},

	// Accessories: farming and economy.
	{Key: "copper_ring", Name: "Медное кольцо", Rank: RankE, Slot: SlotAccessory, Passive: "farm", Value: 0.03},
	{Key: "lucky_charm", Name: "Талисман удачи", Rank: RankD, Slot: SlotAccessory, Passive: "farm", Value: 0.05},
	{Key: "scholar_pendant", Name: "Подвеска книжника", Rank: RankC, Slot: SlotAccessory, Passive: "xp", Value: 0.08},
	{Key: "merchant_seal", Name: "Печать торговца", Rank: RankB, Slot: SlotAccessory, Passive: "farm", Value: 0.12},
	{Key: "star_compass", Name: "Звёздный компас", Rank: RankA, Slot: SlotAccessory, Passive: "xp", Value: 0.18},
	{Key: "guild_signet", Name: "Знак гильдии", Rank: RankS, Slot: SlotAccessory, Passive: "farm", Value: 0.25},
	{Key: "fate_coin", Name: "Монета судьбы", Rank: RankSS, Slot: SlotAccessory, Passive: "farm", Value: 0.35},
	{Key: "origin_crown", Name: "Венец начала", Rank: RankSSS, Slot: SlotAccessory, Passive: "farm", Value: 0.50},
	{Key: "echo_blade", Name: "Клинок эха", Rank: RankB, Slot: SlotWeapon, Passive: "echo", Value: .65},
	{Key: "troll_heart", Name: "Сердце тролля", Rank: RankB, Slot: SlotArmor, Passive: "heal", Value: .12},
	{Key: "hourglass", Name: "Песочные часы", Rank: RankA, Slot: SlotAccessory, Passive: "time", Value: 5},
	{Key: "executioner", Name: "Печать палача", Rank: RankA, Slot: SlotAccessory, Passive: "execute", Value: .45},
}

// relicByKey finds a relic in the catalog.
func relicByKey(key string) (Relic, bool) {
	for _, relic := range relicCatalog {
		if relic.Key == key {
			return relic, true
		}
	}
	return Relic{}, false
}

// rankWeights is the pull table. Higher ranks are rarer, and the sum is 1000 so
// the roll is a single integer comparison.
var rankWeights = []struct {
	Rank   Rank
	Weight int
}{
	{RankE, 380},
	{RankD, 260},
	{RankC, 180},
	{RankB, 100},
	{RankA, 50},
	{RankS, 20},
	{RankSS, 8},
	{RankSSS, 2},
}

// rollRank picks a rank. floor raises the minimum: the pity system uses it so a
// long dry streak cannot continue past the guarantee.
func rollRank(rnd *rand.Rand, floor Rank) Rank {
	total := 0
	for _, entry := range rankWeights {
		if atLeast(entry.Rank, floor) {
			total += entry.Weight
		}
	}
	if total == 0 {
		return RankSSS
	}
	roll := rnd.Intn(total)
	for _, entry := range rankWeights {
		if !atLeast(entry.Rank, floor) {
			continue
		}
		if roll < entry.Weight {
			return entry.Rank
		}
		roll -= entry.Weight
	}
	return RankE
}

// rollRelic picks one relic of the given rank. A rank may have no relic in the
// catalog, so the caller gets the nearest lower rank instead of nothing.
func rollRelic(rnd *rand.Rand, rank Rank) Relic {
	for index := rankIndex(rank); index >= 0; index-- {
		pool := make([]Relic, 0, 8)
		for _, relic := range relicCatalog {
			if relic.Rank == rankOrder[index] {
				pool = append(pool, relic)
			}
		}
		if len(pool) > 0 {
			return pool[rnd.Intn(len(pool))]
		}
	}
	return relicCatalog[0]
}

// HeroClass is the chosen path. Each one bends the run in a different direction.
type HeroClass string

const (
	ClassStrategist HeroClass = "strategist"
	ClassBerserker  HeroClass = "berserker"
	ClassSage       HeroClass = "sage"
	ClassKeeper     HeroClass = "keeper"
	ClassHunter     HeroClass = "hunter"
)

// classBonus is what a path changes. Time is extra seconds per question; the
// rest are multipliers on the matching reward or damage.
type classBonus struct {
	Time        int
	Farm        float64
	Crit        float64
	Negate      float64
	Raid        float64
	ArenaDamage float64
}

var classBonuses = map[HeroClass]classBonus{
	ClassStrategist: {Time: 5, Farm: 0, Crit: 0, Negate: 0, Raid: 0, ArenaDamage: 0},
	ClassBerserker:  {Time: -3, Farm: 0, Crit: 0, Negate: 0, Raid: 0, ArenaDamage: 0.20},
	ClassSage:       {Time: 0, Farm: 0.10, Crit: 0.05, Negate: 0, Raid: 0, ArenaDamage: 0},
	ClassKeeper:     {Time: 3, Farm: 0, Crit: 0, Negate: 0.10, Raid: 0, ArenaDamage: 0},
	ClassHunter:     {Time: 0, Farm: 0, Crit: 0, Negate: 0, Raid: 0.10, ArenaDamage: 0},
}

func bonusFor(class HeroClass) classBonus {
	if bonus, ok := classBonuses[class]; ok {
		return bonus
	}
	return classBonus{}
}

// classCatalog is the client-facing list.
var classCatalog = []struct {
	ID          HeroClass `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
}{
	{ClassStrategist, "Стратег", "+5 секунд на пример, ровный фарм"},
	{ClassBerserker, "Берсерк", "+20% урона на арене, −3 секунды на пример"},
	{ClassSage, "Мудрец", "+10% ауры с фарма, +5% крит"},
	{ClassKeeper, "Хранитель", "+10% поглощения, +3 секунды на пример"},
	{ClassHunter, "Охотник", "+10% урона по боссам и вратам"},
}

// Three correct answers charge the path ability. Cooldown stays zero for the original API field.
var abilities = map[HeroClass]struct {
	Name     string `json:"name"`
	Icon     string `json:"icon"`
	Text     string `json:"text"`
	Cooldown int    `json:"cooldown_seconds"`
}{
	ClassStrategist: {"Прозрение", "visibility", "Текущий пример: +10 секунд, остаются два варианта", 0},
	ClassBerserker:  {"Ярость", "local_fire_department", "Следующий удар наносит двойной урон", 0},
	ClassSage:       {"Анализ", "psychology", "Победная награда даёт двойную ауру и опыт", 0},
	ClassKeeper:     {"Барьер", "shield", "Следующая ошибка не наносит урона", 0},
	ClassHunter:     {"Метка", "gps_fixed", "Следующий удар по боссу наносит двойной урон", 0},
}

func scaledRelic(relic Relic, level int) Relic {
	relic.Value *= 1 + float64(max(0, level-1))*.2
	relic.UpgradeCost = 0
	if level < 5 {
		relic.UpgradeCost = 100 * (rankIndex(relic.Rank) + 1) * (level + 1) * (level + 1)
	}
	rank := rankIndex(relic.Rank) + 1
	relic.CraftAura = 150 * rank * rank
	material := "bone"
	if rankIndex(relic.Rank) >= 4 {
		material = "ember"
	}
	if rankIndex(relic.Rank) >= 6 {
		material = "void"
	}
	relic.CraftCost = map[string]int{material: 6 * rank * rank}
	switch relic.Key {
	case "echo_blade":
		relic.CraftAura = 150 * rank
		relic.CraftCost = map[string]int{"bone": 18}
	case "troll_heart":
		relic.CraftAura = 150 * rank
		relic.CraftCost = map[string]int{"bone": 18}
	case "hourglass":
		relic.CraftAura = 150 * rank
		relic.CraftCost = map[string]int{"ember": 20}
	case "executioner":
		relic.CraftAura = 150 * rank
		relic.CraftCost = map[string]int{"void": 20}
	}
	relic.UpgradeMaterials = map[string]int{}
	if level >= 3 && level < 5 {
		for material := range relic.CraftCost {
			relic.UpgradeMaterials[material] = rank * (level - 2) * 2
		}
	}
	percent := relic.Value * 100
	switch relic.Passive {
	case "damage":
		relic.Description = fmt.Sprintf("Урон +%.0f%%", percent)
	case "crit":
		relic.Description = fmt.Sprintf("Шанс критического удара +%.0f п.п.", percent)
	case "negate":
		relic.Description = fmt.Sprintf("Снижение получаемого урона %.0f%%", percent)
	case "farm":
		relic.Description = fmt.Sprintf("Аура медитации +%.0f%%", percent)
	case "xp":
		relic.Description = fmt.Sprintf("Опыт победных походов +%.0f%%", percent)
	case "echo":
		relic.Description = fmt.Sprintf("Каждый третий верный удар повторяет %.0f%% урона", percent)
		relic.Effect = "echo"
	case "heal":
		relic.Description = fmt.Sprintf("Лечит %.0f%% здоровья после комнаты", percent)
		relic.Effect = "heal"
	case "time":
		relic.Description = fmt.Sprintf("Первый пример похода: +%.0f секунд", relic.Value)
		relic.Effect = "time"
	case "execute":
		relic.Description = fmt.Sprintf("Урон +%.0f%%, когда у врага не больше 35%% здоровья", percent)
		relic.Effect = "execute"
	}
	return relic
}
func relicsCatalog() []Relic {
	result := make([]Relic, 0, len(relicCatalog))
	for _, relic := range relicCatalog {
		result = append(result, scaledRelic(relic, 1))
	}
	return result
}
