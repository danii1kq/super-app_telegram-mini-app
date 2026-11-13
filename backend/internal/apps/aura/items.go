package aura

import (
	"context"

	"github.com/jackc/pgx/v5"
)

type RelicSet struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Keys        []string `json:"keys"`
	Required    int      `json:"required"`
	Active      bool     `json:"active"`
	Equipped    int      `json:"equipped"`
}

var relicSets = []RelicSet{
	{ID: "crypt", Name: "Клятва глубин", Description: "2 предмета: +10% урона", Keys: []string{"bone_rapier", "abyss_scale"}, Required: 2},
	{ID: "forge", Name: "Пепельный легион", Description: "2 предмета: снижение входящего урона +10 п.п.", Keys: []string{"ash_halberd", "ember_carapace"}, Required: 2},
	{ID: "void", Name: "Безлунный обет", Description: "2 предмета: шанс критического удара +8 п.п.", Keys: []string{"eclipse_lance", "night_clock"}, Required: 2},
}

func setsFor(worn []Relic) []RelicSet {
	result := append([]RelicSet(nil), relicSets...)
	for i := range result {
		for _, key := range result[i].Keys {
			for _, item := range worn {
				if item.Key == key {
					result[i].Equipped++
					break
				}
			}
		}
		result[i].Active = result[i].Equipped >= result[i].Required
	}
	return result
}
func activeSetIDs(worn []Relic) []string {
	result := []string{}
	for _, set := range setsFor(worn) {
		if set.Active {
			result = append(result, set.ID)
		}
	}
	return result
}
func hasSet(session *Session, id string) bool {
	for _, set := range session.Sets {
		if set == id {
			return true
		}
	}
	return false
}
func hasWorn(session *Session, key string) bool {
	for _, item := range session.Worn {
		if item.Key == key {
			return true
		}
	}
	return false
}
func decorateRelic(item Relic) Relic {
	for _, set := range relicSets {
		for _, key := range set.Keys {
			if key == item.Key {
				item.SetID = set.ID
				item.SetName = set.Name
				item.SetBonus = set.Description
			}
		}
	}
	switch item.Key {
	case "curse_fang":
		item.Curse = "Ошибка полностью сбрасывает заряд способности"
		item.Lore = "Клинок питается уверенностью охотника. Сомнение оставляет его голодным."
	case "curse_eye":
		item.Curse = "Применение способности тратит 10% максимального здоровья; нужен запас здоровья"
		item.Lore = "Око открывается только в обмен на часть жизненной силы."
	case "curse_codex":
		item.Curse = "Любое восстановление здоровья уменьшено вдвое"
		item.Lore = "Каждое прочитанное имя отдаляет владельца от мира живых."
	case "trophy_crypt":
		item.Lore = "Печать, оставшаяся после первой победы над хранителем склепа."
		item.ConditionHint = "Первая победа над хранителем склепа"
	case "trophy_forge":
		item.Lore = "Первое пламя не гаснет даже в холодной ладони."
		item.ConditionHint = "Первая победа над Пепельным колоссом"
	case "trophy_void":
		item.Lore = "Осколок, подтверждающий возвращение из первой встречи с владыкой разлома."
		item.ConditionHint = "Первая победа над владыкой пустоты"
	case "vow_unbroken":
		item.Lore = "Обет принимает только клинок, ни разу не дрогнувший перед руной."
		item.ConditionHint = "Зачисти склеп без единой ошибки за весь поход"
	case "vow_untouched":
		item.Lore = "Нити плаща сотканы из выдержки того, кто не просил восстановления."
		item.ConditionHint = "Зачисти кузницу без лечения за весь поход"
	case "vow_execution":
		item.Lore = "Последний удар становится приговором, когда охотник вложил в него пробуждение."
		item.ConditionHint = "Добей владыку разлома усиленным ударом способности"
	default:
		switch item.Source {
		case "false_gate":
			item.Lore = "Реликвия вынесена из зала, чей настоящий ранг скрывала ложная печать."
		case "red_gate":
			item.Lore = "На металле ещё остался жар запечатанных красных врат."
		case "hidden_shrine":
			item.Lore = "Дар святилища, сохранившего забытые имена первых охотников."
		default:
			item.Lore = "Осколок прежней эпохи, чьи руны вновь откликнулись на ауру охотника."
		}
	}
	return item
}
func reduceHealing(session *Session, amount int) int {
	if hasWorn(session, "curse_codex") {
		return amount / 2
	}
	return amount
}
func applyHealing(session *Session, amount int) {
	amount = reduceHealing(session, amount)
	before := session.HeroHP
	session.HeroHP = min(session.HeroMaxHP, session.HeroHP+amount)
	if session.HeroHP > before {
		session.HealingUsed = true
	}
}
func curseMistake(session *Session) {
	if hasWorn(session, "curse_fang") {
		session.Charge = 0
		session.AbilityReady = false
	}
}
func curseAbilityCost(session *Session) int {
	if hasWorn(session, "curse_eye") {
		return max(1, session.HeroMaxHP/10)
	}
	return 0
}
func bonusRelics(data *row, session *Session) []Relic {
	result := []Relic{}
	if session.Kind != SessionGate || session.BossHP > 0 || session.HeroHP <= 0 || session.Room < session.Rooms || acceptedEvent(session) {
		return result
	}
	give := func(key string) {
		if data.Progress.Discoveries[key] {
			return
		}
		item, ok := relicByKey(key)
		if !ok {
			return
		}
		data.Progress.Discoveries[key] = true
		result = append(result, scaledRelic(item, 1))
	}
	switch session.Dungeon {
	case "crypt":
		give("trophy_crypt")
		if session.Errors == 0 {
			give("vow_unbroken")
		}
	case "forge":
		give("trophy_forge")
		if !session.HealingUsed {
			give("vow_untouched")
		}
	case "void":
		give("trophy_void")
		if session.EmpoweredFinisher {
			give("vow_execution")
		}
	}
	return result
}

type JournalEntry struct {
	ID         string `json:"id"`
	Key        string `json:"key"`
	Name       string `json:"name"`
	Dungeon    string `json:"dungeon"`
	Hint       string `json:"hint"`
	Discovered bool   `json:"discovered"`
	Revealed   bool   `json:"revealed"`
}

func journalFor(progress Progress) []JournalEntry {
	entries := []JournalEntry{}
	for _, key := range []string{"vow_unbroken", "vow_untouched", "vow_execution"} {
		item, _ := relicByKey(key)
		discovered := progress.Discoveries[key]
		revealed := discovered || progress.BossClears[item.Dungeon] >= 2
		hint := "Условие скрыто. Победи босса этих врат дважды, чтобы прочитать запись"
		if revealed {
			hint = item.ConditionHint
		}
		entries = append(entries, JournalEntry{ID: key, Key: key, Name: item.Name, Dungeon: item.Dungeon, Hint: hint, Discovered: discovered, Revealed: revealed})
	}
	return entries
}
func catalogFor(progress Progress) []Relic {
	items := relicsCatalog()
	for i := range items {
		if items[i].Source == "condition" && !progress.Discoveries[items[i].Key] && progress.BossClears[items[i].Dungeon] < 2 {
			items[i].ConditionHint = "Условие скрыто: изучи журнал после двух побед над боссом"
		}
	}
	return items
}
func favoriteOwned(items []OwnedRelic, keys []string) []OwnedRelic {
	result := []OwnedRelic{}
	for _, key := range keys {
		for _, item := range items {
			if item.Key == key {
				item.Favorite = true
				result = append(result, item)
				break
			}
		}
	}
	return result
}
func (s *Service) Favorite(ctx context.Context, userID int64, key string) (Result, error) {
	result := Result{}
	err := s.pool.InTx(ctx, func(tx pgx.Tx) error {
		data, err := s.locked(ctx, tx, userID)
		if err != nil {
			return err
		}
		var owned bool
		if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM aura_relics WHERE user_id=$1 AND relic_key=$2)`, userID, key).Scan(&owned); err != nil {
			return err
		}
		if !owned {
			return bad("На витрину можно поставить только свою реликвию")
		}
		for i, favorite := range data.Progress.Favorites {
			if favorite == key {
				data.Progress.Favorites = append(data.Progress.Favorites[:i], data.Progress.Favorites[i+1:]...)
				result.Message = "Реликвия убрана с витрины"
				return s.save(ctx, tx, userID, data)
			}
		}
		if len(data.Progress.Favorites) >= 3 {
			return bad("На витрине три места. Сначала убери одну реликвию")
		}
		data.Progress.Favorites = append(data.Progress.Favorites, key)
		result.Message = "Реликвия добавлена на витрину"
		return s.save(ctx, tx, userID, data)
	})
	return s.resultPlayer(ctx, userID, result, err)
}
