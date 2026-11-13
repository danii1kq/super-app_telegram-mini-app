package split

import (
	"context"
	"strconv"
	"testing"
)

func mustID(raw string) int64 {
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		panic(err)
	}
	return id
}

func TestBilateralSettlementPreservesRecipientsAndNeedsAgreement(t *testing.T) {
	s, _ := receiptTestService(t)
	ctx := context.Background()
	g, err := s.CreateGroup(ctx, 1, "Loans", []int64{2, 3})
	if err != nil {
		t.Fatal(err)
	}
	first, err := s.CreateDebt(ctx, 1, g.ID, 2, 1, 400000, "", "ab")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.CreateDebt(ctx, 2, g.ID, 1, 2, 150000, "", "ba"); err != nil {
		t.Fatal(err)
	}
	if _, err = s.CreateDebt(ctx, 3, g.ID, 3, 2, 100000, "", "bc"); err != nil {
		t.Fatal(err)
	}
	pairs, _, err := s.SettlementPlan(ctx, 1, g.ID)
	if err != nil || len(pairs) != 2 {
		t.Fatal(pairs, err)
	}
	var pair SettlementPair
	for _, p := range pairs {
		if p.DebtorID == "1" {
			pair = p
		}
	}
	if pair.CreditorID != "2" || pair.TransferMinor != "250000" || pair.OffsetMinor != "150000" {
		t.Fatal(pair)
	}
	p, err := s.ProposeSettlement(ctx, 1, g.ID, 2, pair.GroupVersion, "settle")
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.RespondSettlement(ctx, 1, mustID(p.ID), "report")
	requireReceiptStatus(t, err, 409)
	p, err = s.RespondSettlement(ctx, 2, mustID(p.ID), "agree")
	if err != nil || p.Status != "agreed" {
		t.Fatal(p, err)
	}
	p, err = s.RespondSettlement(ctx, 1, mustID(p.ID), "report")
	if err != nil || p.Status != "reported" {
		t.Fatal(p, err)
	}
	d, err := s.debtsByID(ctx, 1, first.ID)
	if err != nil || d.Outstanding != 400000 {
		t.Fatal("proposal changed debt", d, err)
	}
	p, err = s.RespondSettlement(ctx, 2, mustID(p.ID), "confirm")
	if err != nil || p.Status != "confirmed" {
		t.Fatal(p, err)
	}
	p, err = s.RespondSettlement(ctx, 2, mustID(p.ID), "confirm")
	if err != nil || p.Status != "confirmed" {
		t.Fatal(p, err)
	}
	debts, err := s.Debts(ctx, 1, g.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range debts {
		if d.CreditorID == 3 {
			if d.Outstanding != 100000 {
				t.Fatal("third recipient changed", d)
			}
		} else if d.Outstanding != 0 {
			t.Fatal("pair not settled", d)
		}
	}
	balances, err := s.Balances(ctx, 1, g.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, b := range balances {
		if b.UserID == 1 && b.BalanceMinor != 0 {
			t.Fatal("balance", b)
		}
	}
}

func TestSettlementStaleAfterPaymentAndLegacyExpense(t *testing.T) {
	s, _ := receiptTestService(t)
	ctx := context.Background()
	g, err := s.CreateGroup(ctx, 1, "Dinner", []int64{2})
	if err != nil {
		t.Fatal(err)
	}
	expense, err := s.AddExpense(ctx, 1, g.ID, 1, "food", "Food", "cafe", 600000, []int64{1, 2})
	if err != nil {
		t.Fatal(err)
	}
	pairs, _, err := s.SettlementPlan(ctx, 1, g.ID)
	if err != nil || len(pairs) != 1 {
		t.Fatal(pairs, err)
	}
	p, err := s.ProposeSettlement(ctx, 1, g.ID, 2, pairs[0].GroupVersion, "first")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.CreateDebt(ctx, 1, g.ID, 1, 2, 100, "", "new"); err != nil {
		t.Fatal(err)
	}
	p, err = s.RespondSettlement(ctx, 2, mustID(p.ID), "agree")
	if err != nil || p.Status != "stale" {
		t.Fatal(p, err)
	}
	pairs, _, err = s.SettlementPlan(ctx, 1, g.ID)
	if err != nil {
		t.Fatal(err)
	}
	p, err = s.ProposeSettlement(ctx, 2, g.ID, 1, pairs[0].GroupVersion, "second")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.RecordExpenseDebt(ctx, 1, g.ID, expense.ID, 2, "track-after-proposal"); err != nil {
		t.Fatal(err)
	}
	p, err = s.RespondSettlement(ctx, 1, mustID(p.ID), "agree")
	if err != nil {
		t.Fatal(err)
	}
	p, err = s.RespondSettlement(ctx, 1, mustID(p.ID), "confirm")
	if err != nil || p.Status != "confirmed" {
		t.Fatal(p, err)
	}
	balances, err := s.Balances(ctx, 1, g.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, b := range balances {
		if b.BalanceMinor != 0 {
			t.Fatal("legacy expense was not settled", b)
		}
	}
}

func TestEqualDebtsSettleAfterBothAgreeWithoutTransfer(t *testing.T) {
	s, _ := receiptTestService(t)
	ctx := context.Background()
	g, err := s.CreateGroup(ctx, 1, "Equal", []int64{2})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.CreateDebt(ctx, 1, g.ID, 1, 2, 100, "", "a"); err != nil {
		t.Fatal(err)
	}
	if _, err = s.CreateDebt(ctx, 2, g.ID, 2, 1, 100, "", "b"); err != nil {
		t.Fatal(err)
	}
	pairs, _, err := s.SettlementPlan(ctx, 1, g.ID)
	if err != nil || len(pairs) != 1 || pairs[0].TransferMinor != "0" {
		t.Fatal(pairs, err)
	}
	p, err := s.ProposeSettlement(ctx, 1, g.ID, 2, pairs[0].GroupVersion, "offset")
	if err != nil {
		t.Fatal(err)
	}
	p, err = s.RespondSettlement(ctx, 2, mustID(p.ID), "agree")
	if err != nil || p.Status != "confirmed" {
		t.Fatal(p, err)
	}
	debts, err := s.Debts(ctx, 1, g.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range debts {
		if d.Outstanding != 0 || len(d.Payments) != 1 || d.Payments[0].Method != "offset" {
			t.Fatal(d)
		}
	}
}
