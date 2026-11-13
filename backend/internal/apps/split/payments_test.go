package split

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
)

func TestPaymentRequiresCreditorAndPreservesBalance(t *testing.T) {
	s, _ := receiptTestService(t)
	ctx := context.Background()
	g, err := s.CreateGroup(ctx, 1, "Dinner", []int64{2, 3})
	if err != nil {
		t.Fatal(err)
	}
	expense, err := s.AddExpense(ctx, 1, g.ID, 1, "meal", "Meal", "cafe", 600000, []int64{1, 2})
	if err != nil {
		t.Fatal(err)
	}
	d, err := s.RecordExpenseDebt(ctx, 1, g.ID, expense.ID, 2, "track")
	if err != nil {
		t.Fatal(err)
	}
	d, err = s.AddPayment(ctx, 2, d.ID, 100000, "sent")
	if err != nil || d.Outstanding != 300000 || d.Payments[0].Status != "pending" {
		t.Fatal(d, err)
	}
	repeated, err := s.AddPayment(ctx, 2, d.ID, 100000, "sent")
	if err != nil || len(repeated.Payments) != 1 {
		t.Fatal(repeated, err)
	}
	_, err = s.AddPayment(ctx, 2, d.ID, 100001, "sent")
	requireReceiptStatus(t, err, 409)
	_, err = s.RespondPayment(ctx, 2, d.ID, d.Payments[0].ID, true)
	requireReceiptStatus(t, err, 403)
	balances, err := s.Balances(ctx, 1, g.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, b := range balances {
		if b.UserID == 1 && b.BalanceMinor != 300000 {
			t.Fatal("pending changed balance", b)
		}
	}
	d, err = s.RespondPayment(ctx, 1, d.ID, d.Payments[0].ID, true)
	if err != nil || d.Outstanding != 200000 {
		t.Fatal(d, err)
	}
	d, err = s.RespondPayment(ctx, 1, d.ID, d.Payments[0].ID, true)
	if err != nil || d.Outstanding != 200000 {
		t.Fatal(d, err)
	}
	balances, err = s.Balances(ctx, 1, g.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, b := range balances {
		if b.UserID == 1 && b.BalanceMinor != 200000 {
			t.Fatal("confirmed missing from balance", b)
		}
	}
	d, err = s.AddPayment(ctx, 2, d.ID, 200000, "rest")
	if err != nil {
		t.Fatal(err)
	}
	d, err = s.RespondPayment(ctx, 1, d.ID, d.Payments[len(d.Payments)-1].ID, false)
	if err != nil || d.Outstanding != 200000 {
		t.Fatal(d, err)
	}
	d, err = s.AddPayment(ctx, 1, d.ID, 200000, "received")
	if err != nil || d.Outstanding != 0 {
		t.Fatal(d, err)
	}
}

func TestConcurrentPaymentsReserveRemaining(t *testing.T) {
	s, _ := receiptTestService(t)
	ctx := context.Background()
	g, err := s.CreateGroup(ctx, 1, "Loan", []int64{2})
	if err != nil {
		t.Fatal(err)
	}
	d, err := s.CreateDebt(ctx, 1, g.ID, 1, 2, 100, "", "loan")
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	var succeeded atomic.Int64
	for _, key := range []string{"one", "two"} {
		wg.Add(1)
		go func(key string) {
			defer wg.Done()
			_, err := s.AddPayment(ctx, 2, d.ID, 100, key)
			if err == nil {
				succeeded.Add(1)
			} else {
				requireReceiptStatus(t, err, 409)
			}
		}(key)
	}
	wg.Wait()
	if succeeded.Load() != 1 {
		t.Fatal("overreserved", succeeded.Load())
	}
}

func TestDebtReplayReturnsActualRemainderAndRejectsAnotherGroup(t *testing.T) {
	s, _ := receiptTestService(t)
	ctx := context.Background()
	g, err := s.CreateGroup(ctx, 1, "Loan", []int64{2})
	if err != nil {
		t.Fatal(err)
	}
	d, err := s.CreateDebt(ctx, 1, g.ID, 1, 2, 100, "Note", "loan")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.AddPayment(ctx, 1, d.ID, 100, "received"); err != nil {
		t.Fatal(err)
	}
	repeated, err := s.CreateDebt(ctx, 1, g.ID, 1, 2, 100, "Note", "loan")
	if err != nil || repeated.Outstanding != 0 {
		t.Fatal("replay reset debt", repeated, err)
	}
	other, err := s.CreateGroup(ctx, 1, "Other", []int64{2})
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.CreateDebt(ctx, 1, other.ID, 1, 2, 100, "Note", "loan")
	requireReceiptStatus(t, err, 409)
	_, err = s.CreateDebt(ctx, 1, g.ID, 1, 2, 100, "Different note", "loan")
	requireReceiptStatus(t, err, 409)
}
