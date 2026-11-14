package split

import (
	"context"
	"testing"
)

func TestReceiptStableItemsAndCombos(t *testing.T) {
	s, _ := receiptTestService(t)
	ctx := context.Background()
	g, err := s.CreateGroup(ctx, 1, "Food", []int64{2})
	if err != nil {
		t.Fatal(err)
	}
	r, err := s.CreateManualReceipt(ctx, 1, g.ID)
	if err != nil {
		t.Fatal(err)
	}
	parent := 0
	r, err = s.UpdateReceipt(ctx, 1, r.ID, 300000, 0, 0, []ReceiptItem{{Name: "Combo", Quantity: 1, AmountMinor: 300000, Units: map[int64]int{2: 1}}, {Name: "Cola", Quantity: 1, ParentPosition: &parent, Units: map[int64]int{1: 1}}}, r.Version)
	if err != nil {
		t.Fatal(err)
	}
	firstID, secondID := r.Items[0].ID, r.Items[1].ID
	if r.Items[1].Units[2] != 1 || r.Items[1].Units[1] != 0 {
		t.Fatalf("combo allocation %+v", r.Items)
	}
	old := r.Version
	r.Items[0].Name = "Updated combo"
	r, err = s.UpdateReceipt(ctx, 1, r.ID, 300000, 0, 0, r.Items, r.Version)
	if err != nil {
		t.Fatal(err)
	}
	if r.Items[0].ID != firstID || r.Items[1].ID != secondID {
		t.Fatal("item IDs changed")
	}
	_, err = s.UpdateReceipt(ctx, 1, r.ID, 300000, 0, 0, r.Items, old)
	requireReceiptStatus(t, err, 409)
	r.Items[0].AllocationMode = "shared"
	r.Items[0].Units = map[int64]int{1: 1, 2: 1}
	r, err = s.UpdateReceipt(ctx, 1, r.ID, 300000, 0, 0, r.Items, r.Version)
	if err != nil {
		t.Fatal(err)
	}
	a := Allocate(r, 1)
	if a.PerPerson[1] != 150000 || a.PerPerson[2] != 150000 || a.Difference != 0 {
		t.Fatalf("shared combo %+v", a)
	}
	r.Items[0].ParentPosition = &parent
	if _, err = s.UpdateReceipt(ctx, 1, r.ID, 300000, 0, 0, r.Items, r.Version); err == nil {
		t.Fatal("accepted cyclic combo")
	}
}
