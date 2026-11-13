package split

import (
	"context"
	"errors"
	"strconv"

	"github.com/danii1kq/srunal/internal/platform/httpapi"
	"github.com/danii1kq/srunal/internal/platform/secure"
	"github.com/jackc/pgx/v5"
)

func touchFinance(ctx context.Context, tx pgx.Tx, groupID int64) error {
	_, err := tx.Exec(ctx, `UPDATE split_groups SET financial_version=financial_version+1 WHERE id=$1`, groupID)
	return err
}

func (s *Service) lockedDebt(ctx context.Context, tx pgx.Tx, userID, debtID int64) (Debt, error) {
	var d Debt
	var sealed []byte
	var key string
	if err := tx.QueryRow(ctx, `SELECT group_id FROM split_debts WHERE id=$1`, debtID).Scan(&d.GroupID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return d, httpapi.NotFound("Долг не найден")
		}
		return d, err
	}
	if err := lockGroup(ctx, tx, d.GroupID); err != nil {
		return d, err
	}
	err := tx.QueryRow(ctx, `SELECT d.id,d.author_id,d.creditor_id,d.debtor_id,d.amount_sealed,d.amount_key_id,d.source_expense_id,d.version FROM split_debts d JOIN split_members m ON m.group_id=d.group_id AND m.user_id=$2 AND m.state='active' WHERE d.id=$1 FOR UPDATE OF d`, debtID, userID).Scan(&d.ID, &d.AuthorID, &d.CreditorID, &d.DebtorID, &sealed, &key, &d.SourceExpenseID, &d.Version)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return d, httpapi.NotFound("Долг не найден")
		}
		return d, err
	}
	if userID != d.CreditorID && userID != d.DebtorID {
		return d, httpapi.NotFound("Долг не найден")
	}
	raw, err := s.content.OpenString(secure.Sealed{Ciphertext: sealed, KeyID: key}, debtAmountAAD(debtID))
	if err != nil {
		return d, err
	}
	d.AmountMinor, err = strconv.ParseInt(raw, 10, 64)
	return d, err
}

func (s *Service) addPaymentTx(ctx context.Context, tx pgx.Tx, d Debt, authorID, amount int64, requestID, status, method string, silent ...bool) (bool, error) {
	if amount <= 0 || amount > 1_000_000_000_000 || requestID == "" || len(requestID) > 128 {
		return false, httpapi.BadRequest("invalid_payment", "Проверьте сумму и ключ возврата")
	}
	if _, err := tx.Exec(ctx, `SELECT id FROM users WHERE id=$1 FOR UPDATE`, authorID); err != nil {
		return false, err
	}
	var existingDebt int64
	var raw []byte
	var key string
	var existingID int64
	err := tx.QueryRow(ctx, `SELECT id,debt_id,amount_sealed,amount_key_id FROM split_debt_payments WHERE author_id=$1 AND client_request_id=$2`, authorID, requestID).Scan(&existingID, &existingDebt, &raw, &key)
	if err == nil {
		previous, err := s.content.OpenString(secure.Sealed{Ciphertext: raw, KeyID: key}, debtPaymentAAD(existingID))
		if err != nil {
			return false, err
		}
		if existingDebt != d.ID || previous != strconv.FormatInt(amount, 10) {
			return false, httpapi.Conflict("idempotency_conflict", "Ключ использован для другого возврата")
		}
		return false, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return false, err
	}
	payments, paid, err := s.paymentsOfTx(ctx, tx, d.ID)
	if err != nil {
		return false, err
	}
	reserved := int64(0)
	for _, p := range payments {
		if p.Status == "pending" {
			reserved += p.AmountMinor
		}
	}
	if amount > d.AmountMinor-paid-reserved {
		return false, httpapi.Conflict("payment_exceeds_remaining", "Возврат превышает свободный остаток. Проверьте ожидающие подтверждения")
	}
	var id int64
	if err = tx.QueryRow(ctx, `INSERT INTO split_debt_payments(debt_id,author_id,amount_sealed,amount_key_id,client_request_id,status,method) VALUES($1,$2,''::bytea,'pending',$3,$4,$5)RETURNING id`, d.ID, authorID, requestID, status, method).Scan(&id); err != nil {
		return false, err
	}
	sealed, err := s.content.SealString(strconv.FormatInt(amount, 10), debtPaymentAAD(id))
	if err != nil {
		return false, err
	}
	if _, err = tx.Exec(ctx, `UPDATE split_debt_payments SET amount_sealed=$2,amount_key_id=$3 WHERE id=$1`, id, sealed.Ciphertext, sealed.KeyID); err != nil {
		return false, err
	}
	if err = s.debtChanged(ctx, tx, d.ID, d.GroupID); err != nil {
		return false, err
	}
	d.Version++
	if status == "confirmed" && paid+amount == d.AmountMinor {
		if _, err = tx.Exec(ctx, `UPDATE split_reminder_agreements SET status='disabled' WHERE debt_id=$1`, d.ID); err != nil {
			return false, err
		}
	}
	if len(silent) > 0 && silent[0] {
		return true, nil
	}
	if status == "confirmed" {
		d.Outstanding = d.AmountMinor - paid - amount
		return true, s.notifyPayment(ctx, tx, d, amount, authorID)
	}
	notice := DebtNotice{Text: "SruSplit: участник отметил перевод " + formatMoney(amount) + ". Подтвердите получение денег в приложении.", ButtonURL: "/apps/split/" + strconv.FormatInt(d.GroupID, 10) + "/debts", ButtonText: "Проверить перевод"}
	return true, s.enqueueNotice(ctx, tx, d.CreditorID, "split.payment_pending", "pending-payment:"+strconv.FormatInt(id, 10), "debt:"+strconv.FormatInt(d.ID, 10), d.Version, notice)
}

func (s *Service) debtChanged(ctx context.Context, tx pgx.Tx, debtID, groupID int64) error {
	if _, err := tx.Exec(ctx, `UPDATE split_debts SET version=version+1 WHERE id=$1`, debtID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE notification_delivery SET state='skipped',last_error='debt changed' WHERE subject=$1 AND state='pending'`, "debt:"+strconv.FormatInt(debtID, 10)); err != nil {
		return err
	}
	return touchFinance(ctx, tx, groupID)
}

func (s *Service) AddPayment(ctx context.Context, authorID, debtID, amount int64, requestID string) (Debt, error) {
	err := s.pool.InTx(ctx, func(tx pgx.Tx) error {
		d, err := s.lockedDebt(ctx, tx, authorID, debtID)
		if err != nil {
			return err
		}
		status := "pending"
		if authorID == d.CreditorID {
			status = "confirmed"
		}
		_, err = s.addPaymentTx(ctx, tx, d, authorID, amount, requestID, status, "transfer")
		return err
	})
	if err != nil {
		return Debt{}, err
	}
	return s.debtsByID(ctx, authorID, debtID)
}

func (s *Service) RespondPayment(ctx context.Context, userID, debtID, paymentID int64, accept bool) (Debt, error) {
	err := s.pool.InTx(ctx, func(tx pgx.Tx) error {
		d, err := s.lockedDebt(ctx, tx, userID, debtID)
		if err != nil {
			return err
		}
		if userID != d.CreditorID {
			return httpapi.Forbidden("Получение подтверждает получатель денег")
		}
		payments, paid, err := s.paymentsOfTx(ctx, tx, debtID)
		if err != nil {
			return err
		}
		var payment *DebtPayment
		for i := range payments {
			if payments[i].ID == paymentID {
				payment = &payments[i]
			}
		}
		if payment == nil {
			return httpapi.NotFound("Перевод не найден")
		}
		status := "rejected"
		if accept {
			status = "confirmed"
		}
		if payment.Status == status {
			return nil
		}
		if payment.Status != "pending" {
			return httpapi.Conflict("payment_resolved", "Перевод уже обработан")
		}
		if accept && payment.AmountMinor > d.AmountMinor-paid {
			return httpapi.Conflict("payment_exceeds_remaining", "Возврат превышает остаток")
		}
		if _, err = tx.Exec(ctx, `UPDATE split_debt_payments SET status=$2 WHERE id=$1`, paymentID, status); err != nil {
			return err
		}
		if err = s.debtChanged(ctx, tx, debtID, d.GroupID); err != nil {
			return err
		}
		d.Version++
		if accept {
			d.Outstanding = d.AmountMinor - paid - payment.AmountMinor
			if d.Outstanding == 0 {
				if _, err = tx.Exec(ctx, `UPDATE split_reminder_agreements SET status='disabled' WHERE debt_id=$1`, debtID); err != nil {
					return err
				}
			}
			return s.notifyPayment(ctx, tx, d, payment.AmountMinor, userID)
		}
		return s.enqueueNotice(ctx, tx, d.DebtorID, "split.payment_rejected", "rejected-payment:"+strconv.FormatInt(paymentID, 10), "debt:"+strconv.FormatInt(debtID, 10), d.Version, DebtNotice{Text: "SruSplit: получатель не подтвердил перевод. Уточните детали друг у друга.", ButtonURL: "/apps/split/" + strconv.FormatInt(d.GroupID, 10) + "/debts", ButtonText: "Открыть"})
	})
	if err != nil {
		return Debt{}, err
	}
	return s.debtsByID(ctx, userID, debtID)
}
