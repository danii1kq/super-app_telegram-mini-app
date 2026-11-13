package split

import (
	"context"
	"github.com/jackc/pgx/v5"
	"strconv"
)

func (s *Service) receiptDebtsTx(ctx context.Context, tx pgx.Tx, r Receipt, expenseID, payerID int64, a ReceiptAllocation) error {
	for debtorID, amount := range a.PerPerson {
		if debtorID == payerID || amount <= 0 {
			continue
		}
		var debtID int64
		if err := tx.QueryRow(ctx, `INSERT INTO split_debts(group_id,author_id,creditor_id,debtor_id,amount_sealed,amount_key_id,source_expense_id,client_request_id)VALUES($1,$2,$3,$4,''::bytea,'pending',$5,$6) RETURNING id`, r.GroupID, r.AuthorID, payerID, debtorID, expenseID, "receipt:"+strconv.FormatInt(r.ID, 10)+":"+strconv.FormatInt(debtorID, 10)).Scan(&debtID); err != nil {
			return err
		}
		sealed, err := s.content.SealString(strconv.FormatInt(amount, 10), debtAmountAAD(debtID))
		if err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `UPDATE split_debts SET amount_sealed=$2,amount_key_id=$3 WHERE id=$1`, debtID, sealed.Ciphertext, sealed.KeyID); err != nil {
			return err
		}
		if err = s.notifyDebt(ctx, tx, Debt{ID: debtID, GroupID: r.GroupID, AuthorID: r.AuthorID, CreditorID: payerID, DebtorID: debtorID, AmountMinor: amount, Outstanding: amount, SourceExpenseID: &expenseID}, r.AuthorID); err != nil {
			return err
		}
	}
	return nil
}
