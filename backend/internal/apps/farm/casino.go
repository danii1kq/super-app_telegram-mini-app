package farm

import (
	"fmt"
	"math"
	"time"

	"github.com/danii1kq/srunal/internal/platform/httpapi"
	"github.com/danii1kq/srunal/internal/platform/secure"
)

const minGameStake = 100
const maxGameStake = 100000
const maxCrashBP = 1000
const crashReturnPercent = 55
const flipFeePercent = 25
const upgradeReturnPercent = 60
const upgradeMaxChanceBP = 4500

type CrashRound struct {
	ReturnPercent int       `json:"return_percent"`
	ID            string    `json:"id"`
	Stake         int       `json:"stake"`
	StartedAt     time.Time `json:"started_at"`
	Status        string    `json:"status"`
	MultiplierBP  int       `json:"multiplier_bp"`
	Payout        int       `json:"payout"`
	BustBP        int       `json:"bust_bp,omitempty"`
}

type GameResult struct {
	Side         string    `json:"side,omitempty"`
	ID           string    `json:"id"`
	Kind         string    `json:"kind"`
	Status       string    `json:"status"`
	Stake        int       `json:"stake"`
	Payout       int       `json:"payout"`
	MultiplierBP int       `json:"multiplier_bp,omitempty"`
	Seed         string    `json:"seed,omitempty"`
	Mutation     string    `json:"mutation,omitempty"`
	ChanceBP     int       `json:"chance_bp,omitempty"`
	At           time.Time `json:"at"`
}

type UpgradeQuote struct {
	SourceValue int     `json:"source_value"`
	TargetValue int     `json:"target_value"`
	ChanceBP    int     `json:"chance_bp"`
	Stake       int     `json:"stake"`
	Target      Harvest `json:"target"`
}

func gameRules() map[string]any {
	return map[string]any{"min_stake": minGameStake, "max_stake": maxGameStake, "crash_max_bp": maxCrashBP, "crash_return_percent": crashReturnPercent, "crash_growth": 0.08, "flip_fee_percent": flipFeePercent, "flip_expiry_seconds": 600, "upgrade_return_percent": upgradeReturnPercent, "upgrade_max_chance_bp": upgradeMaxChanceBP}
}

func crashThreshold(roll int) int       { return crashReturnPercent * 1000000 / (roll + 1) }
func coinflipPayout(stake, fee int) int { return stake * 2 * (100 - fee) / 100 }

func appendGame(state *State, result GameResult) {
	state.GameHistory = append(state.GameHistory, result)
	if len(state.GameHistory) > 20 {
		state.GameHistory = state.GameHistory[len(state.GameHistory)-20:]
	}
}

func validStake(stake int) bool { return stake >= minGameStake && stake <= maxGameStake }

func checkGameRequest(state State, id string) error {
	if len(id) < 16 || len(id) > 80 {
		return httpapi.BadRequest("game_request", "Обнови страницу и повтори действие")
	}
	for _, ch := range id {
		if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '-') {
			return httpapi.BadRequest("game_request", "Некорректный номер запроса")
		}
	}
	if containsID(state.GameRequests, id) {
		return httpapi.BadRequest("game_request", "Этот запрос уже обработан. Обновляем результат")
	}
	return nil
}

func rememberGameRequest(state *State, id string) {
	// shortcut: retain 100 recent request IDs; use a separate ledger for longer retry windows.
	state.GameRequests = append(state.GameRequests, id)
	if len(state.GameRequests) > 100 {
		state.GameRequests = state.GameRequests[len(state.GameRequests)-100:]
	}
}

func crashMultiplier(now, started time.Time) int {
	elapsed := math.Max(0, now.Sub(started).Seconds())
	if elapsed >= math.Log(10)/.08 {
		return maxCrashBP
	}
	return int(math.Floor(math.Exp(elapsed*.08) * 100))
}

func finishCrash(state *State, status string, multiplier int, now time.Time) {
	round := state.Crash
	round.Status = status
	round.MultiplierBP = multiplier
	if status == "cashed" {
		round.Payout = round.Stake * multiplier / 100
		state.Coins += round.Payout
	}
	appendGame(state, GameResult{ID: round.ID, Kind: "crash", Status: status, Stake: round.Stake, Payout: round.Payout, MultiplierBP: multiplier, At: now})
}

func refreshCrash(state *State, now time.Time) bool {
	if state.Crash == nil || state.Crash.Status != "running" {
		return false
	}
	multiplier := crashMultiplier(now, state.Crash.StartedAt)
	if multiplier > state.Crash.BustBP {
		finishCrash(state, "busted", max(100, state.Crash.BustBP+1), now)
		return true
	}
	if multiplier >= maxCrashBP {
		finishCrash(state, "cashed", maxCrashBP, now)
		return true
	}
	return false
}

func publicGameState(state State, now time.Time) State {
	state.GameRequests = nil
	if state.GameHistory == nil {
		state.GameHistory = []GameResult{}
	}
	if state.Crash != nil {
		copy := *state.Crash
		if copy.ReturnPercent == 0 {
			copy.ReturnPercent = 96
		}
		if copy.Status == "running" {
			copy.BustBP = 0
			copy.MultiplierBP = crashMultiplier(now, copy.StartedAt)
		}
		state.Crash = &copy
	}
	return state
}

func plantUpgradeQuote(state State, action Action) (UpgradeQuote, error) {
	bad := func(message string) (UpgradeQuote, error) {
		return UpgradeQuote{}, httpapi.BadRequest("plant_upgrade", message)
	}
	if !validStake(action.Stake) {
		return bad("Ставка должна быть от 100 до 100 000 монет")
	}
	var source *Harvest
	for i := range state.Barn {
		if state.Barn[i].ID == action.Item && state.Barn[i].Quantity > 0 {
			source = &state.Barn[i]
			break
		}
	}
	if source == nil {
		return bad("Выбери растение из амбара")
	}
	crop, ok := species(action.Seed)
	if !ok {
		return bad("Такого растения нет")
	}
	bonus := 0
	for _, mutation := range MutationTypes {
		if mutation.ID == action.Upgrade {
			bonus = mutation.Multiplier
			break
		}
	}
	if bonus == 0 {
		return bad("Выбери мутацию результата")
	}
	target := Harvest{Seed: crop.ID, Quality: "perfect", Mutation: action.Upgrade, Quantity: 1, Value: crop.Price * 3 * bonus}
	market := 100 + state.Upgrades["market"]*5
	sourceValue := source.Value / source.Quantity * market / 100
	targetValue := target.Value * market / 100
	if targetValue <= sourceValue+action.Stake {
		return bad("Выбери результат дороже растения и ставки вместе")
	}
	chance := min(upgradeMaxChanceBP, (sourceValue+action.Stake)*upgradeReturnPercent*100/targetValue)
	if chance < 1 {
		return bad("Для этого результата нужна ставка побольше")
	}
	return UpgradeQuote{SourceValue: sourceValue, TargetValue: targetValue, ChanceBP: chance, Stake: action.Stake, Target: target}, nil
}

func casinoAction(state *State, action Action, now time.Time, roll func(int) (int, error)) (string, error) {
	bad := func(message string) (string, error) { return "", httpapi.BadRequest("farm_game", message) }
	if err := checkGameRequest(*state, action.RequestID); err != nil {
		return "", err
	}
	switch action.Kind {
	case "crash_start":
		if state.Crash != nil && state.Crash.Status == "running" {
			return bad("Сначала заверши текущий разгон")
		}
		if !validStake(action.Stake) {
			return bad("Ставка должна быть от 100 до 100 000 монет")
		}
		if state.Coins < action.Stake {
			return bad("Недостаточно монет")
		}
		r, err := roll(1000000)
		if err != nil {
			return "", err
		}
		id, err := secure.NewToken(16)
		if err != nil {
			return "", err
		}
		state.Coins -= action.Stake
		state.Crash = &CrashRound{ID: id, Stake: action.Stake, StartedAt: now, Status: "running", MultiplierBP: 100, BustBP: crashThreshold(r), ReturnPercent: crashReturnPercent}
		refreshCrash(state, now)
		rememberGameRequest(state, action.RequestID)
		return "Разгон начался. Забрать можно до взрыва, на ×10 сработает автосбор", nil
	case "crash_cashout":
		if state.Crash == nil || state.Crash.ID != action.Item {
			return bad("Этот раунд уже недоступен")
		}
		if state.Crash.Status != "running" {
			return bad("Раунд уже завершён")
		}
		if refreshCrash(state, now) {
			rememberGameRequest(state, action.RequestID)
			return "Раунд завершён по времени сервера", nil
		}
		finishCrash(state, "cashed", crashMultiplier(now, state.Crash.StartedAt), now)
		rememberGameRequest(state, action.RequestID)
		return fmt.Sprintf("Забрано %d монет на ×%.2f", state.Crash.Payout, float64(state.Crash.MultiplierBP)/100), nil
	case "plant_upgrade":
		quote, err := plantUpgradeQuote(*state, action)
		if err != nil {
			return "", err
		}
		if state.Coins < action.Stake {
			return bad("Недостаточно монет")
		}
		r, err := roll(10000)
		if err != nil {
			return "", err
		}
		id, err := secure.NewToken(16)
		if err != nil {
			return "", err
		}
		weight, err := roll(100)
		if err != nil {
			return "", err
		}
		result := GameResult{ID: id, Kind: "upgrade", Status: "lost", Stake: action.Stake, Seed: action.Seed, Mutation: action.Upgrade, ChanceBP: quote.ChanceBP, At: now}
		state.Coins -= action.Stake
		for i, item := range state.Barn {
			if item.ID == action.Item {
				state.Barn[i].Quantity--
				state.Barn[i].Value -= item.Value / item.Quantity
				if state.Barn[i].Quantity == 0 {
					state.Barn = append(state.Barn[:i], state.Barn[i+1:]...)
				}
				break
			}
		}
		if r < quote.ChanceBP {
			plant := quote.Target
			plant.ID = id
			plant.Weight = plantWeight(plant.Seed, weight)
			if plant.Mutation == "giant" {
				plant.Weight *= 3
			}
			state.Barn = append(state.Barn, plant)
			state.Collection[plant.Seed+":"+plant.Mutation] = Discovery{plant.Seed, plant.Quality, plant.Mutation}
			if crop, _ := species(plant.Seed); crop.Rarity == "hybrid" && !containsID(state.Hybrids, plant.Seed) {
				state.Hybrids = append(state.Hybrids, plant.Seed)
			}
			result.Status = "won"
			result.Payout = quote.TargetValue
		}
		appendGame(state, result)
		rememberGameRequest(state, action.RequestID)
		if result.Status == "won" {
			return "Апгрейд удался! Идеальное растение добавлено в амбар", nil
		}
		return "Апгрейд не удался. Растение и ставка израсходованы", nil
	}
	return bad("Неизвестная игра")
}
