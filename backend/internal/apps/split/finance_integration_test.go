package split

import (
	"context"
	"testing"
)

func TestExpenseRepaymentReducesBalanceOnce(t *testing.T) {
	s, _ := receiptTestService(t)
	ctx := context.Background()
	group, err := s.CreateGroup(ctx, 1, "Dinner", []int64{2})
	if err != nil {
		t.Fatal(err)
	}
	expense, err := s.AddExpense(ctx, 1, group.ID, 1, "meal", "Meal", "cafe", 600000, []int64{1, 2})
	if err != nil {
		t.Fatal(err)
	}
	debt, err := s.RecordExpenseDebt(ctx, 1, group.ID, expense.ID, 2, "track")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.AddPayment(ctx, 1, debt.ID, 100000, "partial"); err != nil {
		t.Fatal(err)
	}
	if _, err = s.AddPayment(ctx, 1, debt.ID, 100000, "partial"); err != nil {
		t.Fatal(err)
	}
	balances, err := s.Balances(ctx, 1, group.ID)
	if err != nil {
		t.Fatal(err)
	}
	expected := map[int64]int64{1: 200000, 2: -200000}
	var sum int64
	for _, b := range balances {
		if b.BalanceMinor != expected[b.UserID] {
			t.Fatalf("balance: %+v", b)
		}
		sum += b.BalanceMinor
	}
	if sum != 0 {
		t.Fatalf("sum: %d", sum)
	}
}
