package farm

import (
	"fmt"
	"hash/fnv"
	"sort"
	"time"

	"github.com/danii1kq/srunal/internal/platform/httpapi"
)

// Order is one daily request from the village. It names a plant, optionally a
// minimum quality or an exact mutation, and pays more than the market would.
// The point is to make holding a specific batch worth something, so the barn
// stops being a "sell everything" button.
type Order struct {
	ID       string `json:"id"`
	Seed     string `json:"seed"`
	Quality  string `json:"quality"`
	Mutation string `json:"mutation"`
	Quantity int    `json:"quantity"`
	Coins    int    `json:"coins"`
	XP       int    `json:"xp"`
}

// OrderBoard holds today's orders and which of them are already paid.
type OrderBoard struct {
	Day       string   `json:"day"`
	Orders    []Order  `json:"orders"`
	Fulfilled []string `json:"fulfilled"`
}

// orderTiers keeps the board balanced: one easy order, one mid, one hard. A
// player always has something reachable and something to work toward.
var orderTiers = [][]Order{
	{
		{ID: "easy-carrot", Seed: "carrot", Quantity: 2, Coins: 110, XP: 15},
		{ID: "easy-radish", Seed: "radish", Quantity: 3, Coins: 150, XP: 15},
		{ID: "easy-cucumber", Seed: "cucumber", Quantity: 2, Coins: 140, XP: 15},
	},
	{
		{ID: "mid-strawberry", Seed: "strawberry", Quantity: 2, Coins: 250, XP: 25},
		{ID: "mid-tomato-choice", Seed: "tomato", Quality: "choice", Quantity: 2, Coins: 280, XP: 25},
		{ID: "mid-corn", Seed: "corn", Quantity: 2, Coins: 300, XP: 25},
	},
	{
		{ID: "hard-pumpkin-perfect", Seed: "pumpkin", Quality: "perfect", Quantity: 1, Coins: 520, XP: 40},
		{ID: "hard-tomato-gold", Seed: "tomato", Mutation: "gold", Quantity: 1, Coins: 600, XP: 40},
		{ID: "hard-blueberry", Seed: "blueberry", Quantity: 1, Coins: 380, XP: 30},
		{ID: "hard-carrot-dew", Seed: "carrot", Mutation: "dew", Quantity: 2, Coins: 400, XP: 35},
	},
}

func qualityLevel(quality string) int {
	switch quality {
	case "perfect":
		return 2
	case "choice":
		return 1
	}
	return 0
}

// refreshOrders rebuilds the board when the UTC day changes. Generation is
// deterministic from the player and the date, so the same player sees the same
// orders all day without storing them until they are first requested.
func refreshOrders(state *State, userID int64, now time.Time) {
	day := now.UTC().Format("2006-01-02")
	if state.Orders.Day == day && len(state.Orders.Orders) > 0 {
		return
	}
	rng := orderSeed(userID, day)
	orders := make([]Order, 0, len(orderTiers))
	for _, tier := range orderTiers {
		index := int(rng % uint64(len(tier)))
		rng = rng*6364136223846793005 + 1442695040888963407
		orders = append(orders, tier[index])
	}
	state.Orders = OrderBoard{Day: day, Orders: orders, Fulfilled: []string{}}
}

func orderSeed(userID int64, day string) uint64 {
	h := fnv.New64a()
	fmt.Fprintf(h, "%d|%s", userID, day)
	return h.Sum64()
}

func matchesOrder(item Harvest, order Order) bool {
	if item.Seed != order.Seed {
		return false
	}
	if order.Quality != "" && qualityLevel(item.Quality) < qualityLevel(order.Quality) {
		return false
	}
	if order.Mutation != "" && item.Mutation != order.Mutation {
		return false
	}
	return true
}

// availableFor counts matching plants across the barn without consuming them.
func availableFor(state State, order Order) int {
	total := 0
	for _, item := range state.Barn {
		if matchesOrder(item, order) {
			total += item.Quantity
		}
	}
	return total
}

// Spend the cheapest suitable specimens first so village errands preserve rare finds.
func orderBatches(state State, order Order) []int {
	indices := []int{}
	for i, item := range state.Barn {
		if item.Quantity > 0 && matchesOrder(item, order) {
			indices = append(indices, i)
		}
	}
	sort.SliceStable(indices, func(i, j int) bool {
		a, b := state.Barn[indices[i]], state.Barn[indices[j]]
		return a.Value/a.Quantity < b.Value/b.Quantity
	})
	return indices
}

func orderPayout(state State, order Order) int {
	remaining, value := order.Quantity, 0
	for _, i := range orderBatches(state, order) {
		item := state.Barn[i]
		take := min(remaining, item.Quantity)
		value += item.Value / item.Quantity * take
		remaining -= take
		if remaining == 0 {
			break
		}
	}
	if remaining > 0 {
		return order.Coins
	}
	market := value * (100 + state.Upgrades["market"]*5) / 100
	return max(order.Coins, (market*125+99)/100)
}

func quotedOrders(state State) OrderBoard {
	board := state.Orders
	board.Orders = append([]Order(nil), board.Orders...)
	for i := range board.Orders {
		board.Orders[i].Coins = orderPayout(state, board.Orders[i])
	}
	return board
}

// consumeFor removes exactly the requested amount, keeping each batch's unit
// value consistent so a partial take does not inflate the rest.
func consumeFor(state *State, order Order) bool {
	if availableFor(*state, order) < order.Quantity {
		return false
	}
	remaining := order.Quantity
	for _, i := range orderBatches(*state, order) {
		item := &state.Barn[i]
		take := item.Quantity
		if take > remaining {
			take = remaining
		}
		unit := item.Value / item.Quantity
		item.Quantity -= take
		item.Value -= unit * take
		remaining -= take
		if remaining == 0 {
			break
		}
	}
	kept := state.Barn[:0]
	for _, item := range state.Barn {
		if item.Quantity > 0 {
			kept = append(kept, item)
		}
	}
	state.Barn = kept
	return true
}

func fulfillOrder(state *State, orderID string) (string, error) {
	bad := func(message string) (string, error) { return "", httpapi.BadRequest("farm_order", message) }
	var order *Order
	for i := range state.Orders.Orders {
		if state.Orders.Orders[i].ID == orderID {
			order = &state.Orders.Orders[i]
			break
		}
	}
	if order == nil {
		return bad("Такого заказа нет")
	}
	for _, done := range state.Orders.Fulfilled {
		if done == orderID {
			return bad("Этот заказ уже выполнен")
		}
	}
	payout := orderPayout(*state, *order)
	if !consumeFor(state, *order) {
		return bad("В амбаре не хватает подходящих растений")
	}
	state.Orders.Fulfilled = append(state.Orders.Fulfilled, orderID)
	state.Coins += payout
	state.Sold += payout
	state.XP += order.XP
	crop, _ := species(order.Seed)
	return fmt.Sprintf("Заказ выполнен: %s ×%d. +%d монет", crop.Name, order.Quantity, payout), nil
}
