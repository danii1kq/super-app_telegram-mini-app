package split

import (
	"context"
	"sort"
	"strconv"
	"time"

	"github.com/danii1kq/srunal/internal/platform/httpapi"
	"github.com/danii1kq/srunal/internal/platform/secure"
)

// Category is the spending bucket of an expense or a receipt line.
const (
	CategoryGroceries = "groceries"
	CategoryCafe      = "cafe"
	CategoryTransport = "transport"
	CategoryHome      = "home"
	CategoryFun       = "fun"
	CategoryOther     = "other"
)

// CategoryOther is also the fallback for rows saved before categories existed:
// they must appear in the report, not vanish from it.
func normalizeCategory(value string) string {
	switch value {
	case CategoryGroceries, CategoryCafe, CategoryTransport, CategoryHome, CategoryFun, CategoryOther:
		return value
	}
	return CategoryOther
}

// ReportLine is one purchase as the viewer experienced it.
type ReportLine struct {
	ExpenseID   int64
	GroupID     int64
	Description string
	// SpentAt is the purchase date, not the upload date.
	SpentAt time.Time
	// MyShare is what this person owes for it: their own consumption.
	MyShare int64
	// PaidByMe is the whole amount if this person paid, else zero.
	PaidByMe  int64
	Category  string
	GroupTitle string
}

// Report is one month of a person's own spending.
type Report struct {
	From time.Time
	To   time.Time
	// MySpending is the sum of the viewer's own shares. It is not the sum of
	// receipts they paid for.
	MySpending int64
	// PaidByMe is what the viewer paid, including other people's shares.
	PaidByMe int64
	// ByCategory sums MySpending per category and always adds up to it.
	ByCategory map[string]int64
	Lines      []ReportLine

	// Debt movement in the same month. Lending and repaying are not purchases,
	// so they never enter MySpending.
	LentOut      int64
	ReturnedToMe int64
	ReturnedByMe int64

	// Current outstanding amounts, as of now, not as of the month end.
	OwedToMeNow  int64
	IOweNow      int64
}

// MonthReport builds the report for the month containing `month` in the given
// IANA zone.
//
// The month boundaries are computed in the profile's zone and converted to UTC
// for the query: the server zone and the interface language play no part.
func (s *Service) MonthReport(ctx context.Context, viewerID int64, month string, zone string) (Report, error) {
	location, err := time.LoadLocation(zone)
	if err != nil {
		location = time.UTC
	}

	from, to, err := monthBounds(month, location)
	if err != nil {
		return Report{}, err
	}

	report := Report{
		From:       from,
		To:         to,
		ByCategory: map[string]int64{},
		Lines:      []ReportLine{},
	}

	// Every expense the viewer takes part in, over the whole history: the report
	// total must not be limited by a list page size.
	rows, err := s.pool.Query(ctx,
		`SELECT e.id, e.group_id, e.author_id, e.payer_id, e.description_sealed, e.amount_sealed, e.key_id,
		        e.spent_at, coalesce(e.category, ''), g.title_sealed, g.key_id
		   FROM split_expenses e
		   JOIN split_shares sh ON sh.expense_id = e.id AND sh.user_id = $1
		   JOIN split_groups g ON g.id = e.group_id
		  WHERE e.deleted_at IS NULL
		    AND e.spent_at >= $2 AND e.spent_at < $3
		  ORDER BY e.spent_at, e.id`,
		viewerID, from.UTC(), to.UTC())
	if err != nil {
		return Report{}, err
	}
	defer rows.Close()

	type pending struct {
		line        ReportLine
		shareSealed []byte
		shareKey    string
		amountSealed []byte
		amountKey   string
		authorID    int64
		payerID     int64
		groupSealed []byte
		groupKey    string
	}
	items := make([]pending, 0, 32)

	for rows.Next() {
		var entry pending
		var descriptionSealed, groupSealed []byte
		var groupKey string
		if err := rows.Scan(&entry.line.ExpenseID, &entry.line.GroupID, &entry.authorID, &entry.payerID,
			&descriptionSealed, &entry.amountSealed, &entry.amountKey,
			&entry.line.SpentAt, &entry.line.Category, &groupSealed, &groupKey,
		); err != nil {
			return Report{}, err
		}
		description, err := s.content.OpenString(
			secure.Sealed{KeyID: entry.amountKey, Ciphertext: descriptionSealed},
			expenseDescriptionAAD(entry.line.ExpenseID, entry.authorID),
		)
		if err != nil {
			return Report{}, err
		}
		entry.line.Description = description
		entry.groupSealed = groupSealed
		entry.groupKey = groupKey
		items = append(items, entry)
	}
	if err := rows.Err(); err != nil {
		return Report{}, err
	}

	for _, entry := range items {
		// The viewer's own share, decrypted per row: only the rows this person
		// takes part in are touched.
		var shareSealed []byte
		var shareKey string
		if err := s.pool.QueryRow(ctx,
			`SELECT amount_sealed, key_id FROM split_shares WHERE expense_id = $1 AND user_id = $2`,
			entry.line.ExpenseID, viewerID,
		).Scan(&shareSealed, &shareKey); err != nil {
			continue
		}
		shareRaw, err := s.content.OpenString(
			secure.Sealed{KeyID: shareKey, Ciphertext: shareSealed},
			shareAAD(entry.line.ExpenseID, viewerID),
		)
		if err != nil {
			return Report{}, err
		}
		share, err := strconv.ParseInt(shareRaw, 10, 64)
		if err != nil {
			return Report{}, err
		}
		entry.line.MyShare = share

		if entry.payerID == viewerID {
			amountRaw, err := s.content.OpenString(
				secure.Sealed{KeyID: entry.amountKey, Ciphertext: entry.amountSealed},
				expenseAmountAAD(entry.line.ExpenseID, entry.authorID),
			)
			if err != nil {
				return Report{}, err
			}
			amount, err := strconv.ParseInt(amountRaw, 10, 64)
			if err != nil {
				return Report{}, err
			}
			entry.line.PaidByMe = amount
		}

		if len(entry.groupSealed) > 0 {
			title, err := s.content.OpenString(
				secure.Sealed{KeyID: entry.groupKey, Ciphertext: entry.groupSealed},
				groupTitleAAD(entry.line.GroupID),
			)
			if err == nil {
				entry.line.GroupTitle = title
			}
		}

		entry.line.Category = normalizeCategory(entry.line.Category)
		report.MySpending += entry.line.MyShare
		report.PaidByMe += entry.line.PaidByMe
		report.ByCategory[entry.line.Category] += entry.line.MyShare
		report.Lines = append(report.Lines, entry.line)
	}

	// Debt movement: a loan and a repayment are money moving, not spending.
	lent, returnedIn, returnedOut, err := s.monthDebtMovement(ctx, viewerID, from.UTC(), to.UTC())
	if err != nil {
		return Report{}, err
	}
	report.LentOut = lent
	report.ReturnedToMe = returnedIn
	report.ReturnedByMe = returnedOut

	owedToMe, iOwe, err := s.currentOutstanding(ctx, viewerID)
	if err != nil {
		return Report{}, err
	}
	report.OwedToMeNow = owedToMe
	report.IOweNow = iOwe

	sort.Slice(report.Lines, func(i, j int) bool {
		if !report.Lines[i].SpentAt.Equal(report.Lines[j].SpentAt) {
			return report.Lines[i].SpentAt.Before(report.Lines[j].SpentAt)
		}
		return report.Lines[i].ExpenseID < report.Lines[j].ExpenseID
	})
	return report, nil
}

// monthBounds returns [from, to) in UTC for the month "2026-11" in a zone.
func monthBounds(month string, location *time.Location) (time.Time, time.Time, error) {
	if month == "" {
		now := time.Now().In(location)
		month = now.Format("2006-01")
	}
	start, err := time.ParseInLocation("2006-01", month, location)
	if err != nil {
		return time.Time{}, time.Time{}, httpapi.Validation(
			map[string]string{"month": "Месяц в формате 2026-11"},
			"Проверьте поля",
		)
	}
	return start, start.AddDate(0, 1, 0), nil
}

// monthDebtMovement sums loans and repayments recorded in the month.
func (s *Service) monthDebtMovement(ctx context.Context, viewerID int64, from, to time.Time) (int64, int64, int64, error) {
	var lent int64
	rows, err := s.pool.Query(ctx,
		`SELECT id, creditor_id, debtor_id, amount_sealed, amount_key_id
		   FROM split_debts
		  WHERE source_expense_id IS NULL
		    AND created_at >= $2 AND created_at < $3
		    AND (creditor_id = $1 OR debtor_id = $1)`,
		viewerID, from, to)
	if err != nil {
		return 0, 0, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var id, creditorID, debtorID int64
		var sealed []byte
		var keyID string
		if err := rows.Scan(&id, &creditorID, &debtorID, &sealed, &keyID); err != nil {
			return 0, 0, 0, err
		}
		raw, err := s.content.OpenString(secure.Sealed{KeyID: keyID, Ciphertext: sealed}, debtAmountAAD(id))
		if err != nil {
			return 0, 0, 0, err
		}
		amount, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return 0, 0, 0, err
		}
		// Only the money the viewer lent out counts as "выдано в долг".
		if creditorID == viewerID {
			lent += amount
		}
	}
	if err := rows.Err(); err != nil {
		return 0, 0, 0, err
	}

	var returnedToMe, returnedByMe int64
	payments, err := s.pool.Query(ctx,
		`SELECT p.id, p.amount_sealed, p.amount_key_id, d.creditor_id, d.debtor_id
		   FROM split_debt_payments p
		   JOIN split_debts d ON d.id = p.debt_id
		  WHERE p.paid_at >= $2 AND p.paid_at < $3
		    AND (d.creditor_id = $1 OR d.debtor_id = $1)`,
		viewerID, from, to)
	if err != nil {
		return 0, 0, 0, err
	}
	defer payments.Close()

	for payments.Next() {
		var id, creditorID, debtorID int64
		var sealed []byte
		var keyID string
		if err := payments.Scan(&id, &sealed, &keyID, &creditorID, &debtorID); err != nil {
			return 0, 0, 0, err
		}
		raw, err := s.content.OpenString(secure.Sealed{KeyID: keyID, Ciphertext: sealed}, debtPaymentAAD(id))
		if err != nil {
			return 0, 0, 0, err
		}
		amount, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return 0, 0, 0, err
		}
		if creditorID == viewerID {
			returnedToMe += amount
		}
		if debtorID == viewerID {
			returnedByMe += amount
		}
	}
	return lent, returnedToMe, returnedByMe, payments.Err()
}

// currentOutstanding is the live remainder across every group, not a month-end
// balance.
//
// It comes from the balances, not from the debt rows: an expense already creates
// the obligation, and a debt recorded later only adds a handle for repayments.
// Summing positive balances gives what others owe the viewer; negative balances
// give what the viewer owes.
func (s *Service) currentOutstanding(ctx context.Context, viewerID int64) (int64, int64, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT group_id FROM split_members WHERE user_id = $1 AND state = 'active'`, viewerID)
	if err != nil {
		return 0, 0, err
	}
	defer rows.Close()

	groups := make([]int64, 0, 8)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return 0, 0, err
		}
		groups = append(groups, id)
	}
	if err := rows.Err(); err != nil {
		return 0, 0, err
	}

	var owedToMe, iOwe int64
	for _, groupID := range groups {
		balances, err := s.Balances(ctx, viewerID, groupID)
		if err != nil {
			return 0, 0, err
		}
		for _, balance := range balances {
			if balance.UserID != viewerID {
				continue
			}
			if balance.BalanceMinor > 0 {
				owedToMe += balance.BalanceMinor
			}
			if balance.BalanceMinor < 0 {
				iOwe += -balance.BalanceMinor
			}
		}
	}
	return owedToMe, iOwe, nil
}
