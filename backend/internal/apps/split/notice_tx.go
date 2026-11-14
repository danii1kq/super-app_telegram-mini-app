package split

import (
	"context"
	"strconv"

	"github.com/jackc/pgx/v5"
)

// notifyDebt queues the message for a new debt.
//
// Only the two sides of the debt are notified, and never the person who performed
// the action: sending someone a message about their own click is noise. A debt
// with several lines produces one message, not one per line.
func (s *Service) notifyDebt(ctx context.Context, tx pgx.Tx, debt Debt, actorID int64) error {
	if s.delivery == nil {
		return nil
	}

	names, err := s.displayNames(ctx, tx, []int64{debt.DebtorID, debt.CreditorID})
	if err != nil {
		return err
	}

	for _, recipient := range []int64{debt.DebtorID, debt.CreditorID} {
		if recipient == actorID {
			continue
		}
		notice := noticeForDebt(debt, recipient, names[debt.DebtorID], names[debt.CreditorID], debt.Outstanding)
		dedupe := "debt:" + strconv.FormatInt(debt.ID, 10) + ":" + strconv.FormatInt(recipient, 10)
		subject := "debt:" + strconv.FormatInt(debt.ID, 10)
		if err := s.enqueueNotice(ctx, tx, recipient, "split.debt_created", dedupe, subject, 1, notice); err != nil {
			return err
		}
	}
	return nil
}

// notifyPayment queues the message for a repayment.
//
// The recipient hears the amount and the remainder; the person who recorded it
// does not get an echo of their own action.
func (s *Service) notifyPayment(ctx context.Context, tx pgx.Tx, debt Debt, amountMinor, actorID int64) error {
	if s.delivery == nil {
		return nil
	}

	names, err := s.displayNames(ctx, tx, []int64{debt.DebtorID, debt.CreditorID})
	if err != nil {
		return err
	}

	version := debt.Version
	for _, recipient := range []int64{debt.CreditorID, debt.DebtorID} {
		if recipient == actorID {
			continue
		}

		var text string
		if recipient == debt.CreditorID {
			// The money arrived: this is what they were waiting for.
			if debt.Outstanding <= 0 {
				text = "SruSplit: возврат " + formatMoney(amountMinor) + " учтён. Долг " +
					names[debt.DebtorID] + " закрыт."
			} else {
				text = "SruSplit: возврат " + formatMoney(amountMinor) + " учтён. Осталось получить " +
					formatMoney(debt.Outstanding) + "."
			}
		} else {
			if debt.Outstanding <= 0 {
				text = "SruSplit: ваш долг " + names[debt.CreditorID] + " закрыт."
			} else {
				text = "SruSplit: вы вернули " + formatMoney(amountMinor) + ". Осталось " +
					formatMoney(debt.Outstanding) + "."
			}
		}

		notice := DebtNotice{
			Text:       text,
			ButtonURL:  "/apps/split/" + strconv.FormatInt(debt.GroupID, 10) + "/debts",
			ButtonText: "Открыть долг",
		}
		dedupe := "payment:" + strconv.FormatInt(debt.ID, 10) + ":" + strconv.FormatInt(version, 10) + ":" + strconv.FormatInt(recipient, 10)
		subject := "debt:" + strconv.FormatInt(debt.ID, 10)
		if err := s.enqueueNotice(ctx, tx, recipient, "split.debt_paid", dedupe, subject, version, notice); err != nil {
			return err
		}
	}
	return nil
}

// displayNames reads the public names of the people in one query.
func (s *Service) displayNames(ctx context.Context, tx pgx.Tx, ids []int64) (map[int64]string, error) {
	rows, err := tx.Query(ctx,
		`SELECT id, public_name FROM users WHERE id = ANY($1)`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[int64]string, len(ids))
	for rows.Next() {
		var id int64
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			return nil, err
		}
		out[id] = name
	}
	return out, rows.Err()
}
