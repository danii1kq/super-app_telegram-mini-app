package split

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
)

func TestCollaborativeReceiptConsensusAndInvite(t *testing.T) {
	s, _ := receiptTestService(t)
	ctx := context.Background()
	g, err := s.CreateGroup(ctx, 1, "Dinner", []int64{2})
	if err != nil {
		t.Fatal(err)
	}
	r, err := s.CreateManualReceipt(ctx, 1, g.ID)
	if err != nil {
		t.Fatal(err)
	}
	parent := 0
	r, err = s.UpdateReceipt(ctx, 1, r.ID, 720000, 0, 0, []ReceiptItem{{Name: "Doner", Quantity: 1, AmountMinor: 234996}, {Name: "Cola", Quantity: 1, ParentPosition: &parent}, {Name: "Doner", Quantity: 1, AmountMinor: 195002}, {Name: "Combo", Quantity: 1, AmountMinor: 290002}}, r.Version)
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.SaveReceipt(ctx, 1, r.ID, 1, "bypass", "Dinner", "cafe")
	requireReceiptStatus(t, err, 409)
	r, err = s.ReceiptAction(ctx, 1, r.ID, r.Version, "collect")
	if err != nil {
		t.Fatal(err)
	}
	token, err := s.CreateReceiptInvite(ctx, 1, r.ID)
	if err != nil {
		t.Fatal(err)
	}
	info, err := s.ResolveReceiptInvite(ctx, 3, token, false)
	if err != nil || info.Joined || info.ReceiptID != "" {
		t.Fatal(info, err)
	}
	_, err = s.Receipt(ctx, 3, r.ID)
	requireReceiptStatus(t, err, 404)
	info, err = s.ResolveReceiptInvite(ctx, 3, token, true)
	if err != nil || !info.Joined {
		t.Fatal(info, err)
	}
	for index, user := range []int64{1, 2, 3} {
		position := index
		if index > 0 {
			position++
		}
		r, err = s.ClaimReceiptItem(ctx, user, r.ID, r.Items[position].ID, r.Version, 1)
		if err != nil {
			t.Fatal(err)
		}
	}
	if r.Items[1].Units[1] != 1 {
		t.Fatal("combo child not assigned")
	}
	r, err = s.SubmitReceipt(ctx, 1, r.ID, r.Version, ReceiptReview{PayerID: 1, ClientRequestID: "dinner", Description: "Dinner", Category: "cafe"})
	if err != nil {
		t.Fatal(err)
	}
	expenses, err := s.Expenses(ctx, 1, g.ID)
	if err != nil || len(expenses) != 0 {
		t.Fatal("unconfirmed expense", expenses, err)
	}
	r, err = s.RespondReceipt(ctx, 2, r.ID, r.Version, "question", "Check my item")
	if err != nil {
		t.Fatal(err)
	}
	r, err = s.RespondReceipt(ctx, 3, r.ID, r.Version, "accepted", "")
	if err != nil || r.Status != "review" {
		t.Fatal(r, err)
	}
	r, err = s.ReceiptAction(ctx, 1, r.ID, r.Version, "reopen")
	if err != nil || len(r.Approvals) != 0 {
		t.Fatal("old approvals survived", r, err)
	}
	r, err = s.SubmitReceipt(ctx, 1, r.ID, r.Version, ReceiptReview{PayerID: 1, ClientRequestID: "dinner", Description: "Dinner", Category: "cafe"})
	if err != nil {
		t.Fatal(err)
	}
	r, err = s.RespondReceipt(ctx, 2, r.ID, r.Version, "accepted", "")
	if err != nil {
		t.Fatal(err)
	}
	r, err = s.RespondReceipt(ctx, 3, r.ID, r.Version, "accepted", "")
	if err != nil || r.Status != "saved" {
		t.Fatal(r, err)
	}
	expenses, err = s.Expenses(ctx, 1, g.ID)
	if err != nil || len(expenses) != 1 || expenses[0].Shares[1] != 234996 || expenses[0].Shares[2] != 195002 || expenses[0].Shares[3] != 290002 {
		t.Fatal(expenses, err)
	}
	debts, err := s.Debts(ctx, 1, g.ID)
	if err != nil || len(debts) != 2 {
		t.Fatal(debts, err)
	}
	if err = s.RevokeReceiptInvites(ctx, 1, r.ID); err != nil {
		t.Fatal(err)
	}
	_, err = s.ResolveReceiptInvite(ctx, 3, token, false)
	requireReceiptStatus(t, err, 404)
}

func TestReceiptListReturnsOnlyAuthorizedSummaries(t *testing.T) {
	s, _ := receiptTestService(t)
	ctx := context.Background()
	g, err := s.CreateGroup(ctx, 1, "List", []int64{2})
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := s.CreateManualReceipt(ctx, 1, g.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.UpdateReceipt(ctx, 1, receipt.ID, 12300, 0, 0, []ReceiptItem{{Name: "Private item", Quantity: 1, AmountMinor: 12300}}, receipt.Version); err != nil {
		t.Fatal(err)
	}
	mux := receiptTestMux(t, s)
	for _, user := range []int{1, 3} {
		r := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/split/groups/%d/receipts", g.ID), nil)
		r.Header.Set("Authorization", fmt.Sprintf("Bearer test-%d", user))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		if user == 3 {
			if w.Code != 404 {
				t.Fatal(w.Code, w.Body.String())
			}
			continue
		}
		if w.Code != 200 {
			t.Fatal(w.Code, w.Body.String())
		}
		var data struct {
			Receipts []map[string]any `json:"receipts"`
		}
		if err = json.Unmarshal(w.Body.Bytes(), &data); err != nil {
			t.Fatal(err)
		}
		if len(data.Receipts) != 1 || data.Receipts[0]["total_minor"] != "12300" {
			t.Fatal(data)
		}
		if _, ok := data.Receipts[0]["items"]; ok {
			t.Fatal("list fetched full private receipt")
		}
	}
}

func TestConcurrentClaimKeepsLastUnitAndSharedPizza(t *testing.T) {
	s, _ := receiptTestService(t)
	ctx := context.Background()
	g, err := s.CreateGroup(ctx, 1, "Dinner", []int64{2, 3})
	if err != nil {
		t.Fatal(err)
	}
	r, err := s.CreateManualReceipt(ctx, 1, g.ID)
	if err != nil {
		t.Fatal(err)
	}
	r, err = s.UpdateReceipt(ctx, 1, r.ID, 600000, 0, 0, []ReceiptItem{{Name: "Burger", Quantity: 1, AmountMinor: 200000}, {Name: "Pizza", Quantity: 1, AmountMinor: 400000, AllocationMode: "shared"}}, r.Version)
	if err != nil {
		t.Fatal(err)
	}
	r, err = s.ReceiptAction(ctx, 1, r.ID, r.Version, "collect")
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	var success atomic.Int64
	for _, user := range []int64{2, 3} {
		wg.Add(1)
		go func(user int64) {
			defer wg.Done()
			_, err := s.ClaimReceiptItem(ctx, user, r.ID, r.Items[0].ID, r.Version, 1)
			if err == nil {
				success.Add(1)
			} else {
				requireReceiptStatus(t, err, 409)
			}
		}(user)
	}
	wg.Wait()
	if success.Load() != 1 {
		t.Fatal("overbooked item", success.Load())
	}
	r, err = s.Receipt(ctx, 1, r.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, user := range []int64{1, 2, 3} {
		r, err = s.ClaimReceiptItem(ctx, user, r.ID, r.Items[1].ID, r.Version, 1)
		if err != nil {
			t.Fatal(err)
		}
	}
	a := Allocate(r, 1)
	if a.Difference != 0 || a.PerPerson[1] != 133334 {
		t.Fatal(a)
	}
}
