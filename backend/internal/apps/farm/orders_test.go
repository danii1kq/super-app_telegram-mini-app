package farm

import (
	"testing"
	"time"
)

func boardFor(t *testing.T, userID int64, now time.Time) State {
	t.Helper()
	s := initialState(now)
	refreshOrders(&s, userID, now)
	return s
}

func TestOrdersAreStableWithinADayAndChangeAcrossDays(t *testing.T) {
	now := time.Date(2026, 11, 12, 9, 0, 0, 0, time.UTC)
	first := boardFor(t, 7, now)
	later := boardFor(t, 7, now.Add(6*time.Hour))
	if len(first.Orders.Orders) != len(orderTiers) {
		t.Fatalf("expected %d orders, got %d", len(orderTiers), len(first.Orders.Orders))
	}
	for i := range first.Orders.Orders {
		if first.Orders.Orders[i].ID != later.Orders.Orders[i].ID {
			t.Fatal("the board changed inside the same day")
		}
	}
	// A different day must be able to produce a different board; with three
	// tiers and a handful of options, at least one order should differ across
	// a week of days.
	changed := false
	for day := 1; day <= 7 && !changed; day++ {
		other := boardFor(t, 7, now.AddDate(0, 0, day))
		for i := range other.Orders.Orders {
			if other.Orders.Orders[i].ID != first.Orders.Orders[i].ID {
				changed = true
				break
			}
		}
	}
	if !changed {
		t.Fatal("the board never varies across a week")
	}
}

func TestOrdersArePerPlayer(t *testing.T) {
	now := time.Date(2026, 11, 12, 9, 0, 0, 0, time.UTC)
	seen := map[string]bool{}
	for user := int64(1); user <= 40; user++ {
		s := boardFor(t, user, now)
		seen[orderSignature(s)] = true
	}
	if len(seen) < 2 {
		t.Fatal("every player received the same board")
	}
}

func orderSignature(s State) string {
	out := ""
	for _, order := range s.Orders.Orders {
		out += order.ID + ";"
	}
	return out
}

func TestOrderMatchingRespectsQualityAndMutation(t *testing.T) {
	order := Order{Seed: "tomato", Quality: "choice", Quantity: 1}
	if matchesOrder(Harvest{Seed: "tomato", Quality: "normal"}, order) {
		t.Fatal("normal quality satisfied a choice order")
	}
	if !matchesOrder(Harvest{Seed: "tomato", Quality: "choice"}, order) {
		t.Fatal("choice quality did not satisfy a choice order")
	}
	if !matchesOrder(Harvest{Seed: "tomato", Quality: "perfect"}, order) {
		t.Fatal("perfect quality must satisfy a choice order")
	}
	if matchesOrder(Harvest{Seed: "carrot", Quality: "perfect"}, order) {
		t.Fatal("a different species satisfied the order")
	}

	gold := Order{Seed: "tomato", Mutation: "gold", Quantity: 1}
	if matchesOrder(Harvest{Seed: "tomato", Mutation: "none"}, gold) {
		t.Fatal("an unmutated plant satisfied a gold order")
	}
	if !matchesOrder(Harvest{Seed: "tomato", Mutation: "gold"}, gold) {
		t.Fatal("a gold plant did not satisfy a gold order")
	}
}

func TestFulfillOrderConsumesExactlyAndPaysOnce(t *testing.T) {
	now := time.Date(2026, 11, 12, 9, 0, 0, 0, time.UTC)
	s := boardFor(t, 3, now)
	// Force a known order so the assertions are exact.
	s.Orders.Orders = []Order{{ID: "test-carrot", Seed: "carrot", Quantity: 2, Coins: 110, XP: 15}}
	s.Barn = []Harvest{
		{ID: "a", Seed: "carrot", Quality: "normal", Mutation: "none", Quantity: 3, Value: 90},
		{ID: "b", Seed: "tomato", Quality: "normal", Mutation: "none", Quantity: 1, Value: 48},
	}
	coins, xp := s.Coins, s.XP

	message, err := fulfillOrder(&s, "test-carrot")
	if err != nil {
		t.Fatal(err)
	}
	if message == "" {
		t.Fatal("no confirmation message")
	}
	if s.Coins != coins+110 || s.XP != xp+15 {
		t.Fatal("reward was not paid exactly")
	}
	if len(s.Barn) != 2 || s.Barn[0].Quantity != 1 || s.Barn[0].Value != 30 {
		t.Fatalf("barn not reduced by exactly two units: %+v", s.Barn[0])
	}
	if s.Barn[1].Seed != "tomato" {
		t.Fatal("an unrelated batch was touched")
	}
	if _, err := fulfillOrder(&s, "test-carrot"); err == nil {
		t.Fatal("the same order paid twice")
	}
}

func TestFulfillOrderRejectsWhenShort(t *testing.T) {
	now := time.Date(2026, 11, 12, 9, 0, 0, 0, time.UTC)
	s := boardFor(t, 4, now)
	s.Orders.Orders = []Order{{ID: "test-pumpkin", Seed: "pumpkin", Quality: "perfect", Quantity: 1, Coins: 520}}
	s.Barn = []Harvest{{ID: "a", Seed: "pumpkin", Quality: "choice", Mutation: "none", Quantity: 1, Value: 120}}
	coins := s.Coins
	if _, err := fulfillOrder(&s, "test-pumpkin"); err == nil {
		t.Fatal("a choice pumpkin satisfied a perfect order")
	}
	if s.Coins != coins || len(s.Barn) != 1 {
		t.Fatal("a rejected order changed the state")
	}
	if _, err := fulfillOrder(&s, "missing-order"); err == nil {
		t.Fatal("an unknown order was accepted")
	}
}

func TestFulfillOrderDropsEmptyBatches(t *testing.T) {
	now := time.Date(2026, 11, 12, 9, 0, 0, 0, time.UTC)
	s := boardFor(t, 5, now)
	s.Orders.Orders = []Order{{ID: "test-radish", Seed: "radish", Quantity: 2, Coins: 150}}
	s.Barn = []Harvest{{ID: "only", Seed: "radish", Quality: "normal", Mutation: "none", Quantity: 2, Value: 76}}
	if _, err := fulfillOrder(&s, "test-radish"); err != nil {
		t.Fatal(err)
	}
	if len(s.Barn) != 0 {
		t.Fatalf("an emptied batch stayed in the barn: %+v", s.Barn)
	}
}

func TestOrderBoardSurvivesNormalization(t *testing.T) {
	now := time.Date(2026, 11, 12, 9, 0, 0, 0, time.UTC)
	s := initialState(now)
	refreshOrders(&s, 9, now)
	// A save written before orders existed has an empty board; normalization
	// plus refresh must fill it rather than leaving the client with nothing.
	s.Orders = OrderBoard{}
	refreshOrders(&s, 9, now)
	if len(s.Orders.Orders) == 0 {
		t.Fatal("an empty board was not rebuilt")
	}
	if s.Orders.Fulfilled == nil {
		t.Fatal("fulfilled list must not be nil")
	}
}

func TestOrderKeepsExpensiveSpecimensWhenCheaperPlantsExist(t *testing.T) {
	s := initialState(time.Now())
	s.Orders.Orders = []Order{{ID: "carrots", Seed: "carrot", Quantity: 2, Coins: 110}}
	s.Barn = []Harvest{
		{ID: "gold", Seed: "carrot", Quality: "perfect", Mutation: "gold", Quantity: 2, Value: 960},
		{ID: "normal", Seed: "carrot", Quality: "normal", Mutation: "none", Quantity: 2, Value: 64},
	}
	coins := s.Coins
	if _, err := fulfillOrder(&s, "carrots"); err != nil {
		t.Fatal(err)
	}
	if len(s.Barn) != 1 || s.Barn[0].ID != "gold" || s.Barn[0].Value != 960 || s.Coins != coins+110 || s.Sold != 110 {
		t.Fatalf("order consumed a rare specimen or lost sale revenue: %+v", s)
	}
}

func TestOrderQuotePaysPremiumOverActualMarketValue(t *testing.T) {
	s := initialState(time.Now())
	s.Upgrades["market"] = 5
	s.Orders.Orders = []Order{{ID: "gold", Seed: "tomato", Mutation: "gold", Quantity: 1, Coins: 600}}
	s.Barn = []Harvest{{ID: "perfect", Seed: "tomato", Quality: "perfect", Mutation: "gold", Quantity: 1, Value: 720}}
	quote := quotedOrders(s)
	if quote.Orders[0].Coins != 1125 || s.Orders.Orders[0].Coins != 600 {
		t.Fatal("quote missed market premium or changed base order")
	}
	coins := s.Coins
	if _, err := fulfillOrder(&s, "gold"); err != nil {
		t.Fatal(err)
	}
	if s.Coins != coins+1125 || s.Sold != 1125 || len(s.Barn) != 0 {
		t.Fatal("paid less than the displayed quote")
	}
}
