package split

import "testing"

// TestMandatoryExample covers the case from the brief: you and Dima, one pizza
// for both and one burger for Dima only.
func TestMandatoryExample(t *testing.T) {
	const (
		me   int64 = 1
		dima int64 = 2
	)

	receipt := Receipt{
		TotalMinor: 600000,
		Items: []ReceiptItem{
			{ID: 1, Name: "Пицца", Quantity: 1, AmountMinor: 400000, Units: map[int64]int{me: 1, dima: 1}},
			{ID: 2, Name: "Бургер", Quantity: 1, AmountMinor: 200000, Units: map[int64]int{dima: 1}},
		},
	}

	allocation := Allocate(receipt, me)

	if allocation.TotalMinor != 600000 {
		t.Fatalf("total: got %d, want 600000", allocation.TotalMinor)
	}
	if allocation.PerPerson[me] != 200000 {
		t.Fatalf("my share: got %d, want 200000", allocation.PerPerson[me])
	}
	if allocation.PerPerson[dima] != 400000 {
		t.Fatalf("Dima's share: got %d, want 400000", allocation.PerPerson[dima])
	}
	if allocation.OwesToPayer[dima] != 400000 {
		t.Fatalf("Dima owes: got %d, want 400000", allocation.OwesToPayer[dima])
	}
	if _, paysSelf := allocation.OwesToPayer[me]; paysSelf {
		t.Fatal("the payer must not owe themselves")
	}
	if allocation.Difference != 0 {
		t.Fatalf("difference: got %d, want 0", allocation.Difference)
	}
	if allocation.SumShares != 600000 {
		t.Fatalf("sum of shares: got %d, want 600000", allocation.SumShares)
	}
}

// TestPersonalItemNotShared is the rule that personal purchases never spread
// over the group.
func TestPersonalItemNotShared(t *testing.T) {
	receipt := Receipt{
		TotalMinor: 200000,
		Items: []ReceiptItem{
			{ID: 1, Name: "Бургер", Quantity: 1, AmountMinor: 200000, Units: map[int64]int{2: 1}},
		},
	}
	allocation := Allocate(receipt, 1)

	if allocation.PerPerson[1] != 0 {
		t.Fatalf("payer took nothing, share must be 0, got %d", allocation.PerPerson[1])
	}
	if allocation.PerPerson[2] != 200000 {
		t.Fatalf("Dima's share: got %d, want 200000", allocation.PerPerson[2])
	}
}

// TestUnitsSplitNotPeople covers two drinks to one person and one to another:
// the split follows units, not headcount.
func TestUnitsSplitNotPeople(t *testing.T) {
	receipt := Receipt{
		TotalMinor: 300000,
		Items: []ReceiptItem{
			{ID: 1, Name: "Напитки", Quantity: 3, AmountMinor: 300000, Units: map[int64]int{1: 2, 2: 1}},
		},
	}
	allocation := Allocate(receipt, 1)

	if allocation.PerPerson[1] != 200000 {
		t.Fatalf("two of three units: got %d, want 200000", allocation.PerPerson[1])
	}
	if allocation.PerPerson[2] != 100000 {
		t.Fatalf("one of three units: got %d, want 100000", allocation.PerPerson[2])
	}
}

// TestRoundingKeepsTotal checks the invariant on an amount that does not divide
// evenly: the shares must still add up to the receipt.
func TestRoundingKeepsTotal(t *testing.T) {
	receipt := Receipt{
		TotalMinor: 100000,
		Items: []ReceiptItem{
			{ID: 1, Name: "Пицца", Quantity: 1, AmountMinor: 100000, Units: map[int64]int{1: 1, 2: 1, 3: 1}},
		},
	}
	allocation := Allocate(receipt, 1)

	if allocation.SumShares != 100000 {
		t.Fatalf("sum: got %d, want 100000", allocation.SumShares)
	}
	if allocation.Difference != 0 {
		t.Fatalf("difference: got %d, want 0", allocation.Difference)
	}
	if allocation.PerPerson[1]+allocation.PerPerson[2]+allocation.PerPerson[3] != 100000 {
		t.Fatal("shares must add up to the receipt")
	}
	// 100000 / 3 = 33334 + 33333 + 33333, never 33333.33.
	if allocation.PerPerson[1] != 33334 {
		t.Fatalf("largest remainder goes first: got %d, want 33334", allocation.PerPerson[1])
	}
}

// TestDiscountAndServiceOnlyForParticipants: a person who took nothing must not
// pay for a service fee.
func TestDiscountAndServiceOnlyForParticipants(t *testing.T) {
	receipt := Receipt{
		TotalMinor: 110000,
		Service:    10000,
		Items: []ReceiptItem{
			{ID: 1, Name: "Пицца", Quantity: 1, AmountMinor: 100000, Units: map[int64]int{1: 1, 2: 1}},
		},
	}
	allocation := Allocate(receipt, 1)

	if _, present := allocation.PerPerson[3]; present {
		t.Fatal("a person who took nothing must not appear in the split")
	}
	if allocation.SumShares != 110000 {
		t.Fatalf("service must be included: got %d, want 110000", allocation.SumShares)
	}
	if allocation.PerPerson[1] != 55000 || allocation.PerPerson[2] != 55000 {
		t.Fatalf("service split evenly: got %d and %d", allocation.PerPerson[1], allocation.PerPerson[2])
	}
}

// TestUnallocatedItemIsReported: a line nobody took must be visible, not
// silently dropped.
func TestUnallocatedItemIsReported(t *testing.T) {
	receipt := Receipt{
		TotalMinor: 300000,
		Items: []ReceiptItem{
			{ID: 1, Name: "Пицца", Quantity: 1, AmountMinor: 100000, Units: map[int64]int{1: 1}},
			{ID: 2, Name: "Забыли", Quantity: 1, AmountMinor: 200000, Units: map[int64]int{}},
		},
	}
	allocation := Allocate(receipt, 1)

	if len(allocation.Unallocated) != 1 || allocation.Unallocated[0] != 2 {
		t.Fatalf("unallocated: got %v, want [2]", allocation.Unallocated)
	}
	if allocation.Difference != 200000 {
		t.Fatalf("difference: got %d, want 200000", allocation.Difference)
	}
}

// TestSplitByWeightsIsExact checks the low-level divider on uneven weights.
func TestSplitByWeightsIsExact(t *testing.T) {
	shares, err := splitByWeights(1000, map[int64]int64{1: 2, 2: 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if shares[1]+shares[2] != 1000 {
		t.Fatalf("shares must add up: got %d", shares[1]+shares[2])
	}
	if shares[1] != 667 {
		t.Fatalf("weight 2 takes the remainder: got %d, want 667", shares[1])
	}
}

// TestDetectImageMime rejects a file that only claims to be an image.
func TestDetectImageMime(t *testing.T) {
	if got := detectImageMime([]byte("not an image at all")); got != "" {
		t.Fatalf("plain text must be rejected, got %q", got)
	}
	png := []byte("\x89PNG\r\n\x1a\n0000")
	if got := detectImageMime(png); got != "image/png" {
		t.Fatalf("png magic: got %q", got)
	}
	jpeg := []byte{0xFF, 0xD8, 0xFF, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	if got := detectImageMime(jpeg); got != "image/jpeg" {
		t.Fatalf("jpeg magic: got %q", got)
	}
}

func TestReceiptWeightsAvoidOverflow(t *testing.T) {
	shares, err := splitByWeights(1_000_000_000_000, map[int64]int64{2: 900_000_000_000, 10: 100_000_000_000})
	if err != nil || shares[2] != 900_000_000_000 || shares[10] != 100_000_000_000 {
		t.Fatalf("large weights %v %v", shares, err)
	}
	discount, err := splitByWeights(-1, map[int64]int64{2: 1, 10: 1})
	if err != nil || discount[2] != -1 || discount[10] != 0 {
		t.Fatalf("discount %v %v", discount, err)
	}
	if _, err = splitByWeights(1, map[int64]int64{1: 9223372036854775807, 2: 1}); err == nil {
		t.Fatal("accepted overflowing weights")
	}
}

func TestReceiptServiceWithRoundedZeroShare(t *testing.T) {
	allocation := Allocate(Receipt{TotalMinor: 2, Service: 1, Items: []ReceiptItem{{ID: 1, Name: "Tiny", Quantity: 1, AmountMinor: 1, Units: map[int64]int{1: 1, 2: 1, 3: 1}}}}, 1)
	if allocation.Difference != 0 || allocation.SumShares != 2 || allocation.PerPerson[1] != 2 {
		t.Fatalf("service must not disappear with zero-rounded shares: %#v", allocation)
	}
}
