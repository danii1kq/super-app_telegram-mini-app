package aura

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

type Quest struct {
	ID              string         `json:"id"`
	Name            string         `json:"name"`
	Description     string         `json:"description"`
	Progress        int            `json:"progress"`
	Goal            int            `json:"goal"`
	Claimed         bool           `json:"claimed"`
	RewardAura      int            `json:"reward_aura"`
	RewardMaterials map[string]int `json:"reward_materials"`
}

func questsFor(data row) []Quest {
	p := data.Progress
	list := []Quest{
		{ID: "first_clear", Name: "Первый контракт", Description: "Зачисти любое подземелье", Progress: p.Clears, Goal: 1, RewardAura: 250, RewardMaterials: map[string]int{"bone": 5}},
		{ID: "rune_reader", Name: "Читатель рун", Description: "Реши 30 примеров правильно", Progress: p.Correct, Goal: 30, RewardAura: 350, RewardMaterials: map[string]int{"bone": 8}},
		{ID: "rank_d", Name: "Экзамен ранга D", Description: "Достигни 5 уровня и зачисти склеп 3 раза", Progress: min(data.Level/5, p.BossClears["crypt"]/3), Goal: 1, RewardAura: 600, RewardMaterials: map[string]int{"bone": 10}},
		{ID: "rank_c", Name: "Экзамен ранга C", Description: "Достигни 10 уровня и зачисти кузницу 3 раза", Progress: min(data.Level/10, p.BossClears["forge"]/3), Goal: 1, RewardAura: 1200, RewardMaterials: map[string]int{"ember": 10}},
		{ID: "rank_b", Name: "Экзамен ранга B", Description: "Достигни 20 уровня и зачисти разлом 5 раз", Progress: min(data.Level/20, p.BossClears["void"]/5), Goal: 1, RewardAura: 2000, RewardMaterials: map[string]int{"void": 10}},
		{ID: "rank_a", Name: "Экзамен ранга A", Description: "Достигни 30 уровня и зачисти 40 подземелий", Progress: min(data.Level/30, p.Clears/40), Goal: 1, RewardAura: 3500, RewardMaterials: map[string]int{"void": 20}},
		{ID: "rank_s", Name: "Экзамен ранга S", Description: "Достигни 40 уровня и зачисти 80 подземелий", Progress: min(data.Level/40, p.Clears/80), Goal: 1, RewardAura: 6000, RewardMaterials: map[string]int{"void": 30}},
	}
	for i := range list {
		list[i].Claimed = p.Claimed[list[i].ID]
		list[i].Progress = min(list[i].Progress, list[i].Goal)
	}
	return list
}
func (s *Service) QuestClaim(ctx context.Context, userID int64, id string) (Result, error) {
	result := Result{}
	err := s.pool.InTx(ctx, func(tx pgx.Tx) error {
		data, err := s.locked(ctx, tx, userID)
		if err != nil {
			return err
		}
		for _, quest := range questsFor(data) {
			if quest.ID != id {
				continue
			}
			if quest.Claimed || quest.Progress < quest.Goal {
				return bad("Задание ещё не выполнено или награда уже получена")
			}
			data.Progress.Claimed[id] = true
			data.Aura += int64(quest.RewardAura)
			data.TotalAura += int64(quest.RewardAura)
			for key, amount := range quest.RewardMaterials {
				data.Progress.Materials[key] += amount
			}
			ranks := map[string]string{"rank_d": "D", "rank_c": "C", "rank_b": "B", "rank_a": "A", "rank_s": "S"}
			if rank := ranks[id]; rank != "" && rankIndex(Rank(rank)) > rankIndex(Rank(data.Progress.Rank)) {
				data.Progress.Rank = rank
			}
			result.Message = "Система: награда получена"
			result.AuraGain = int64(quest.RewardAura)
			result.MaterialsGain = quest.RewardMaterials
			return s.save(ctx, tx, userID, data)
		}
		return bad("Такого задания нет")
	})
	return s.resultPlayer(ctx, userID, result, err)
}
func (s *Service) Difficulty(ctx context.Context, userID int64, difficulty string) (Result, error) {
	if !validDifficulty(difficulty) {
		return Result{}, bad("Выбери доступную сложность")
	}
	err := s.pool.InTx(ctx, func(tx pgx.Tx) error {
		data, err := s.locked(ctx, tx, userID)
		if err != nil {
			return err
		}
		if data.Session != nil && s.now().Before(data.Session.ExpiresAt) {
			return bad("Сложность меняется между походами")
		}
		data.Progress.Difficulty = difficulty
		return s.save(ctx, tx, userID, data)
	})
	return s.resultPlayer(ctx, userID, Result{Message: "Сложность следующих походов изменена"}, err)
}
func (s *Service) RelicUpgrade(ctx context.Context, userID int64, key string) (Result, error) {
	result := Result{}
	err := s.pool.InTx(ctx, func(tx pgx.Tx) error {
		data, err := s.locked(ctx, tx, userID)
		if err != nil {
			return err
		}
		relic, ok := relicByKey(key)
		if !ok {
			return bad("Такой реликвии нет")
		}
		var level int
		if err = tx.QueryRow(ctx, `SELECT level FROM aura_relics WHERE user_id=$1 AND relic_key=$2 FOR UPDATE`, userID, key).Scan(&level); err != nil {
			if err == pgx.ErrNoRows {
				return bad("Этой реликвии у тебя нет")
			}
			return err
		}
		if level >= 5 {
			return bad("Реликвия уже пробуждена до максимума")
		}
		upgrade := scaledRelic(relic, level)
		cost := upgrade.UpgradeCost
		if data.Aura < int64(cost) {
			return bad("Недостаточно ауры для пробуждения")
		}
		for material, amount := range upgrade.UpgradeMaterials {
			if data.Progress.Materials[material] < amount {
				return bad("Для пробуждения не хватает материалов подземелья")
			}
		}
		data.Aura -= int64(cost)
		for material, amount := range upgrade.UpgradeMaterials {
			data.Progress.Materials[material] -= amount
		}
		if _, err = tx.Exec(ctx, `UPDATE aura_relics SET level=level+1 WHERE user_id=$1 AND relic_key=$2`, userID, key); err != nil {
			return err
		}
		result.Message = fmt.Sprintf("%s: пробуждение %d", relic.Name, level+1)
		return s.save(ctx, tx, userID, data)
	})
	return s.resultPlayer(ctx, userID, result, err)
}
func (s *Service) RelicCraft(ctx context.Context, userID int64, key string) (Result, error) {
	result := Result{}
	err := s.pool.InTx(ctx, func(tx pgx.Tx) error {
		data, err := s.locked(ctx, tx, userID)
		if err != nil {
			return err
		}
		relic, ok := relicByKey(key)
		if !ok {
			return bad("Такой реликвии нет")
		}
		recipe := scaledRelic(relic, 1)
		if data.Aura < int64(recipe.CraftAura) {
			return bad("Недостаточно ауры для изготовления")
		}
		for material, amount := range recipe.CraftCost {
			if data.Progress.Materials[material] < amount {
				return bad("Не хватает материалов подземелья")
			}
		}
		var exists bool
		if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM aura_relics WHERE user_id=$1 AND relic_key=$2)`, userID, key).Scan(&exists); err != nil {
			return err
		}
		if exists {
			return bad("Реликвия уже есть. Используй пробуждение")
		}
		data.Aura -= int64(recipe.CraftAura)
		for material, amount := range recipe.CraftCost {
			data.Progress.Materials[material] -= amount
		}
		if _, err = s.grantRelic(ctx, tx, userID, relic); err != nil {
			return err
		}
		result.Message = "Изготовлено: " + relic.Name
		result.Relic = &recipe
		return s.save(ctx, tx, userID, data)
	})
	return s.resultPlayer(ctx, userID, result, err)
}
func (s *Service) Route(ctx context.Context, userID int64, route, requestID string) (Result, error) {
	result := Result{}
	err := s.pool.InTx(ctx, func(tx pgx.Tx) error {
		data, err := s.locked(ctx, tx, userID)
		if err != nil {
			return err
		}
		if err = s.checkRequest(ctx, tx, userID, requestID); err != nil {
			return err
		}
		session := data.Session
		if session == nil || session.Phase != "choice" {
			return bad("Сначала зачисти комнату")
		}
		now := s.now()
		if !now.Before(session.ExpiresAt) {
			result = s.finishRun(&data, session, false, now)
		} else {
			switch route {
			case "normal", "elite", "shrine", "chest", "retreat":
			default:
				return bad("Такого пути нет")
			}
			if route == "retreat" {
				result = s.finishRun(&data, session, false, now)
				result.Message = "Ты покинул врата. Материалы сохранены"
			} else {
				if route == "shrine" && data.Aura < 40 {
					return bad("Святилище требует 40 ауры")
				}
				if route == "chest" && session.HeroHP <= 20 {
					return bad("Сундук требует больше 20 здоровья")
				}
				session.Room++
				if route == "shrine" {
					data.Aura -= 40
					session.HeroHP = min(session.HeroMaxHP, session.HeroHP+session.HeroMaxHP/3)
				}
				if route == "chest" {
					session.HeroHP -= 20
					d, _ := dungeonByID(session.Dungeon)
					data.Progress.Materials[d.Material] += 3
					result.MaterialsGain = map[string]int{d.Material: 3}
				}
				if err = s.enterRoom(session, route, now); err != nil {
					return err
				}
				result.Message = "Система: следующий зал"
			}
		}
		if err = s.save(ctx, tx, userID, data); err != nil {
			return err
		}
		return s.record(ctx, tx, userID, string(session.Kind), requestID, 0, 0, 0, 0)
	})
	return s.resultPlayer(ctx, userID, result, err)
}
func (s *Service) Ability(ctx context.Context, userID int64, requestID string) (Result, error) {
	result := Result{}
	err := s.pool.InTx(ctx, func(tx pgx.Tx) error {
		data, err := s.locked(ctx, tx, userID)
		if err != nil {
			return err
		}
		if err = s.checkRequest(ctx, tx, userID, requestID); err != nil {
			return err
		}
		session := data.Session
		if session == nil || session.Phase != "combat" || session.Charge < 3 || session.HeroPath == "" {
			return bad("Способность заряжается тремя верными ответами")
		}
		now := s.now()
		if !now.Before(session.QuestionDeadline) || !now.Before(session.ExpiresAt) {
			return bad("Время на этот пример уже вышло")
		}
		switch session.HeroPath {
		case ClassStrategist:
			choices := []int{session.Question.Answer}
			for _, option := range session.Question.Options {
				if option != session.Question.Answer {
					choices = append(choices, option)
					break
				}
			}
			s.rnd.Shuffle(len(choices), func(i, j int) { choices[i], choices[j] = choices[j], choices[i] })
			session.Question.Options = choices
			session.QuestionDeadline = session.QuestionDeadline.Add(10 * time.Second)
		case ClassBerserker, ClassHunter:
			if session.Empowered {
				return bad("Усиленный удар уже готов")
			}
			session.Empowered = true
		case ClassKeeper:
			if session.Shield {
				return bad("Барьер уже готов")
			}
			session.Shield = true
		case ClassSage:
			if session.DoubleReward {
				return bad("Награда уже усилена")
			}
			session.DoubleReward = true
		default:
			return bad("Выбери путь охотника")
		}
		session.Charge = 0
		session.AbilityReady = false
		session.Ability = true
		result.Message = "Способность активирована: " + abilities[session.HeroPath].Name
		if err = s.save(ctx, tx, userID, data); err != nil {
			return err
		}
		return s.record(ctx, tx, userID, string(session.Kind), requestID, 0, 0, 0, 0)
	})
	return s.resultPlayer(ctx, userID, result, err)
}
