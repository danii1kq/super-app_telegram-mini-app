package split

import "testing"

func TestSplitDistributesRemainder(t *testing.T) {
	shares, err := Split(10000, []int64{3, 1, 2})
	if err != nil {
		t.Fatalf("Split: %v", err)
	}
	// 100.00 KZT over three: 33.34 + 33.33 + 33.33, remainder to the lowest id.
	want := map[int64]int64{1: 3334, 2: 3333, 3: 3333}
	for id, expected := range want {
		if shares[id] != expected {
			t.Fatalf("share of %d = %d, want %d", id, shares[id], expected)
		}
	}
	if !CheckShares(10000, shares) {
		t.Fatal("shares must add up to the expense")
	}
}

func TestSplitIsStableRegardlessOfInputOrder(t *testing.T) {
	first, err := Split(10000, []int64{3, 1, 2})
	if err != nil {
		t.Fatalf("Split: %v", err)
	}
	second, err := Split(10000, []int64{1, 2, 3})
	if err != nil {
		t.Fatalf("Split: %v", err)
	}
	for id, share := range first {
		if second[id] != share {
			t.Fatalf("share of %d changed with input order", id)
		}
	}
}

func TestSplitSingleParticipant(t *testing.T) {
	shares, err := Split(4999, []int64{42})
	if err != nil {
		t.Fatalf("Split: %v", err)
	}
	if shares[42] != 4999 {
		t.Fatalf("single participant must take the whole amount, got %d", shares[42])
	}
}

func TestSplitRejectsInvalidInput(t *testing.T) {
	if _, err := Split(0, []int64{1}); err != ErrInvalidAmount {
		t.Fatalf("zero amount: got %v", err)
	}
	if _, err := Split(-100, []int64{1}); err != ErrInvalidAmount {
		t.Fatalf("negative amount: got %v", err)
	}
	if _, err := Split(100, nil); err != ErrNoParticipants {
		t.Fatalf("no participants: got %v", err)
	}
	if _, err := Split(100, []int64{1, 1}); err != ErrDuplicateParticipant {
		t.Fatalf("duplicate participant: got %v", err)
	}
}

// A group balance must sum to zero: that is the property the UI shows.
func TestBalancesSumToZero(t *testing.T) {
	// Two expenses: A paid 10000 split three ways, B paid 3000 split two ways.
	first, err := Split(10000, []int64{1, 2, 3})
	if err != nil {
		t.Fatalf("Split: %v", err)
	}
	second, err := Split(3000, []int64{1, 2})
	if err != nil {
		t.Fatalf("Split: %v", err)
	}

	paid := map[int64]int64{1: 10000, 2: 3000, 3: 0}
	shares := map[int64]int64{}
	for id, share := range first {
		shares[id] += share
	}
	for id, share := range second {
		shares[id] += share
	}

	var total int64
	for id := range paid {
		total += BalanceOf(paid[id], shares[id])
	}
	if total != 0 {
		t.Fatalf("balances must sum to zero, got %d", total)
	}
}

func TestCheckSharesRejectsMismatch(t *testing.T) {
	if CheckShares(100, map[int64]int64{1: 60, 2: 39}) {
		t.Fatal("a total that does not match must be rejected")
	}
	if CheckShares(100, map[int64]int64{1: 120, 2: -20}) {
		t.Fatal("a negative share must be rejected")
	}
}
