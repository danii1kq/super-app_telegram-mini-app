package split

import "sort"

// Split divides amountMinor between participants so the shares add up to the
// original amount exactly.
//
// Integer arithmetic only: q = amount / n for everyone, and the first r
// participants in a stable order (ascending user id) get one extra minor unit.
// 10000 over three people is 3334 + 3333 + 3333, never 3333.33.
func Split(amountMinor int64, participantIDs []int64) (map[int64]int64, error) {
	if len(participantIDs) == 0 {
		return nil, ErrNoParticipants
	}
	if amountMinor <= 0 {
		return nil, ErrInvalidAmount
	}

	// Stable order: the same input always produces the same remainder split.
	ids := make([]int64, len(participantIDs))
	copy(ids, participantIDs)
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })

	// Duplicates would silently drop a share.
	for index := 1; index < len(ids); index++ {
		if ids[index] == ids[index-1] {
			return nil, ErrDuplicateParticipant
		}
	}

	count := int64(len(ids))
	base := amountMinor / count
	remainder := amountMinor % count

	shares := make(map[int64]int64, len(ids))
	for index, id := range ids {
		share := base
		if int64(index) < remainder {
			share++
		}
		shares[id] = share
	}
	return shares, nil
}

// BalanceOf is what a participant paid minus what their shares cost. A positive
// balance means the group owes them; the sum over a group is always zero.
func BalanceOf(paidMinor int64, shareMinor int64) int64 {
	return paidMinor - shareMinor
}

// CheckShares verifies the invariant before a commit: every participant has a
// non-negative share and the shares add up to the expense.
func CheckShares(amountMinor int64, shares map[int64]int64) bool {
	var total int64
	for _, share := range shares {
		if share < 0 {
			return false
		}
		total += share
	}
	return total == amountMinor
}
