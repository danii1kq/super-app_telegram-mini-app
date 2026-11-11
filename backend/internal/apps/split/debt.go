package split

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/danii1kq/srunal/internal/platform/httpapi"
	"github.com/danii1kq/srunal/internal/platform/secure"
)

// CreateDebt records an obligation without a receipt.
//
// A manual debt has no expense behind it, so it does add to the group balance.
// The author is recorded and shown: the other side has not confirmed anything.
func (s *Service) CreateDebt(ctx context.Context, authorID, groupID, creditorID, debtorID int64, amountMinor int64, note, clientRequestID string) (Debt, error) {
	if clientRequestID == "" {
		return Debt{}, httpapi.BadRequest("missing_request_id", "Не передан client_request_id")
	}
	if err := s.requireMember(ctx, authorID, groupID); err != nil {
		return Debt{}, err
	}
	if creditorID == debtorID {
		return Debt{}, httpapi.Validation(map[string]string{"debtor_id": "Должник и получатель совпадают"}, "Проверьте поля")
	}
	if amountMinor <= 0 {
		return Debt{}, httpapi.Validation(map[string]string{"amount_minor": "Сумма должна быть больше нуля"}, "Проверьте поля")
	}
	if err := s.requireMember(ctx, creditorID, groupID); err != nil {
		return Debt{}, httpapi.Validation(map[string]string{"creditor_id": "Получатель не входит в группу"}, "Проверьте поля")
	}
	if err := s.requireMember(ctx, debtorID, groupID); err != nil {
		return Debt{}, httpapi.Validation(map[string]string{"debtor_id": "Должник не входит в группу"}, "Проверьте поля")
	}

	var debt Debt
	err := s.pool.InTx(ctx, func(tx pgx.Tx) error {
		// Idempotency: the same key with the same body returns the stored row; a
		// different body under the same key is a conflict, not a silent accept.
		var existingID, existingCreditor, existingDebtor int64
		var existingSealed []byte
		var existingKey string
		err := tx.QueryRow(ctx,
			`SELECT id, creditor_id, debtor_id, amount_sealed, amount_key_id
			   FROM split_debts WHERE author_id = $1 AND client_request_id = $2`,
			authorID, clientRequestID,
		).Scan(&existingID, &existingCreditor, &existingDebtor, &existingSealed, &existingKey)
		if err == nil {
			existingRaw, err := s.content.OpenString(
				secure.Sealed{KeyID: existingKey, Ciphertext: existingSealed}, debtAmountAAD(existingID),
			)
			if err != nil {
				return err
			}
			existingAmount, err := strconv.ParseInt(existingRaw, 10, 64)
			if err != nil {
				return err
			}
			if existingCreditor != creditorID || existingDebtor != debtorID || existingAmount != amountMinor {
				return httpapi.Conflict("idempotency_conflict", "Этот ключ уже использован с другими данными")
			}
			debt.ID = existingID
			return nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}

		if err := tx.QueryRow(ctx,
			`INSERT INTO split_debts (group_id, author_id, creditor_id, debtor_id, amount_sealed, amount_key_id, note_sealed, note_key_id, client_request_id)
			 VALUES ($1, $2, $3, $4, ''::bytea, 'pending', NULL, NULL, $5)
			 RETURNING id, created_at`,
			groupID, authorID, creditorID, debtorID, clientRequestID,
		).Scan(&debt.ID, &debt.CreatedAt); err != nil {
			return err
		}

		amountSealed, err := s.content.SealString(strconv.FormatInt(amountMinor, 10), debtAmountAAD(debt.ID))
		if err != nil {
			return err
		}
		var noteSealed []byte
		var noteKey *string
		if note != "" {
			sealed, err := s.content.SealString(note, debtNoteAAD(debt.ID))
			if err != nil {
				return err
			}
			noteSealed = sealed.Ciphertext
			noteKey = &sealed.KeyID
		}
		if _, err := tx.Exec(ctx,
			`UPDATE split_debts SET amount_sealed = $2, amount_key_id = $3, note_sealed = $4, note_key_id = $5 WHERE id = $1`,
			debt.ID, amountSealed.Ciphertext, amountSealed.KeyID, noteSealed, noteKey,
		); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return Debt{}, err
	}

	debt.GroupID = groupID
	debt.AuthorID = authorID
	debt.CreditorID = creditorID
	debt.DebtorID = debtorID
	debt.AmountMinor = amountMinor
	debt.Note = note
	debt.Outstanding = amountMinor
	return debt, nil
}

// RecordExpenseDebt links an obligation to an expense that already carries it.
//
// This is the "записать в долг" step. The expense already put the share into the
// balance, so the debt row is a handle for repayments and must not add the amount
// again. The unique index on (source_expense_id, debtor_id) makes a repeat a
// no-op instead of a second debt.
func (s *Service) RecordExpenseDebt(ctx context.Context, authorID, groupID, expenseID, debtorID int64, clientRequestID string) (Debt, error) {
	if clientRequestID == "" {
		return Debt{}, httpapi.BadRequest("missing_request_id", "Не передан client_request_id")
	}
	if err := s.requireMember(ctx, authorID, groupID); err != nil {
		return Debt{}, err
	}

	expense, err := s.Expense(ctx, authorID, groupID, expenseID)
	if err != nil {
		return Debt{}, err
	}
	share, ok := expense.Shares[debtorID]
	if !ok || share <= 0 {
		return Debt{}, httpapi.Validation(map[string]string{"debtor_id": "У этого участника нет доли в расходе"}, "Проверьте поля")
	}
	if debtorID == expense.PayerID {
		return Debt{}, httpapi.Validation(map[string]string{"debtor_id": "Плательщик не должен сам себе"}, "Проверьте поля")
	}

	var debt Debt
	err = s.pool.InTx(ctx, func(tx pgx.Tx) error {
		var existingID int64
		err := tx.QueryRow(ctx,
			`SELECT id FROM split_debts WHERE source_expense_id = $1 AND debtor_id = $2`,
			expenseID, debtorID,
		).Scan(&existingID)
		if err == nil {
			debt.ID = existingID
			return nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}

		if err := tx.QueryRow(ctx,
			`INSERT INTO split_debts (group_id, author_id, creditor_id, debtor_id, amount_sealed, amount_key_id, source_expense_id, client_request_id)
			 VALUES ($1, $2, $3, $4, ''::bytea, 'pending', $5, $6)
			 RETURNING id, created_at`,
			groupID, authorID, expense.PayerID, debtorID, expenseID, clientRequestID,
		).Scan(&debt.ID, &debt.CreatedAt); err != nil {
			return err
		}

		amountSealed, err := s.content.SealString(strconv.FormatInt(share, 10), debtAmountAAD(debt.ID))
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx,
			`UPDATE split_debts SET amount_sealed = $2, amount_key_id = $3 WHERE id = $1`,
			debt.ID, amountSealed.Ciphertext, amountSealed.KeyID,
		); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return Debt{}, err
	}

	debt.GroupID = groupID
	debt.AuthorID = authorID
	debt.CreditorID = expense.PayerID
	debt.DebtorID = debtorID
	debt.AmountMinor = share
	debt.SourceExpenseID = &expenseID
	debt.Outstanding = share
	return debt, nil
}

// Debts lists the group's obligations with their repayments.
func (s *Service) Debts(ctx context.Context, viewerID, groupID int64) ([]Debt, error) {
	if err := s.requireMember(ctx, viewerID, groupID); err != nil {
		return nil, err
	}

	rows, err := s.pool.Query(ctx,
		`SELECT id, author_id, creditor_id, debtor_id, amount_sealed, amount_key_id,
		        note_sealed, note_key_id, source_expense_id, created_at
		   FROM split_debts WHERE group_id = $1 ORDER BY created_at DESC, id DESC`, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]Debt, 0, 8)
	for rows.Next() {
		var debt Debt
		var amountSealed []byte
		var amountKey string
		var noteSealed []byte
		var noteKey *string
		if err := rows.Scan(&debt.ID, &debt.AuthorID, &debt.CreditorID, &debt.DebtorID,
			&amountSealed, &amountKey, &noteSealed, &noteKey, &debt.SourceExpenseID, &debt.CreatedAt,
		); err != nil {
			return nil, err
		}
		debt.GroupID = groupID

		amountRaw, err := s.content.OpenString(secure.Sealed{KeyID: amountKey, Ciphertext: amountSealed}, debtAmountAAD(debt.ID))
		if err != nil {
			return nil, err
		}
		amount, err := strconv.ParseInt(amountRaw, 10, 64)
		if err != nil {
			return nil, err
		}
		debt.AmountMinor = amount

		if len(noteSealed) > 0 && noteKey != nil {
			note, err := s.content.OpenString(secure.Sealed{KeyID: *noteKey, Ciphertext: noteSealed}, debtNoteAAD(debt.ID))
			if err != nil {
				return nil, err
			}
			debt.Note = note
		}

		payments, paid, err := s.paymentsOf(ctx, debt.ID)
		if err != nil {
			return nil, err
		}
		debt.Payments = payments
		debt.PaidMinor = paid
		debt.Outstanding = amount - paid
		out = append(out, debt)
	}
	return out, rows.Err()
}

func (s *Service) paymentsOf(ctx context.Context, debtID int64) ([]DebtPayment, int64, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, author_id, amount_sealed, amount_key_id, paid_at
		   FROM split_debt_payments WHERE debt_id = $1 ORDER BY paid_at, id`, debtID)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	out := make([]DebtPayment, 0, 4)
	total := int64(0)
	for rows.Next() {
		var payment DebtPayment
		var sealed []byte
		var keyID string
		if err := rows.Scan(&payment.ID, &payment.AuthorID, &sealed, &keyID, &payment.PaidAt); err != nil {
			return nil, 0, err
		}
		raw, err := s.content.OpenString(secure.Sealed{KeyID: keyID, Ciphertext: sealed}, debtPaymentAAD(payment.ID))
		if err != nil {
			return nil, 0, err
		}
		amount, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return nil, 0, err
		}
		payment.AmountMinor = amount
		total += amount
		out = append(out, payment)
	}
	return out, total, rows.Err()
}

// AddPayment records a full or partial repayment.
//
// The debt row is locked for the duration of the transaction, so two concurrent
// repayments cannot together exceed the debt. The client request id makes a
// retry after a network error return the stored payment instead of a second one.
func (s *Service) AddPayment(ctx context.Context, authorID, debtID, amountMinor int64, clientRequestID string) (Debt, error) {
	if clientRequestID == "" {
		return Debt{}, httpapi.BadRequest("missing_request_id", "Не передан client_request_id")
	}
	if amountMinor <= 0 {
		return Debt{}, httpapi.Validation(map[string]string{"amount_minor": "Возврат должен быть больше нуля"}, "Проверьте поля")
	}

	err := s.pool.InTx(ctx, func(tx pgx.Tx) error {
		var groupID, creditorID, debtorID int64
		var amountSealed []byte
		var amountKey string
		if err := tx.QueryRow(ctx,
			`SELECT group_id, creditor_id, debtor_id, amount_sealed, amount_key_id
			   FROM split_debts WHERE id = $1 FOR UPDATE`, debtID,
		).Scan(&groupID, &creditorID, &debtorID, &amountSealed, &amountKey); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return httpapi.NotFound("Долг не найден")
			}
			return err
		}
		// Only the two sides of the debt may record money moving.
		if authorID != creditorID && authorID != debtorID {
			return httpapi.NotFound("Долг не найден")
		}

		// Idempotency first: a retry must not add a second repayment.
		var existingID int64
		err := tx.QueryRow(ctx,
			`SELECT id FROM split_debt_payments WHERE author_id = $1 AND client_request_id = $2`,
			authorID, clientRequestID,
		).Scan(&existingID)
		if err == nil {
			return nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}

		amountRaw, err := s.content.OpenString(secure.Sealed{KeyID: amountKey, Ciphertext: amountSealed}, debtAmountAAD(debtID))
		if err != nil {
			return err
		}
		debtAmount, err := strconv.ParseInt(amountRaw, 10, 64)
		if err != nil {
			return err
		}

		// Ciphertext cannot be summed in SQL, so the running total is decrypted
		// here inside the same transaction that holds the debt row lock.
		_, paidTotal, err := s.paymentsOfTx(ctx, tx, debtID)
		if err != nil {
			return err
		}

		if paidTotal+amountMinor > debtAmount {
			return httpapi.Validation(
				map[string]string{"amount_minor": "Возврат больше остатка долга"},
				"Проверьте поля",
			)
		}

		var paymentID int64
		if err := tx.QueryRow(ctx,
			`INSERT INTO split_debt_payments (debt_id, author_id, amount_sealed, amount_key_id, client_request_id)
			 VALUES ($1, $2, ''::bytea, 'pending', $3) RETURNING id`,
			debtID, authorID, clientRequestID,
		).Scan(&paymentID); err != nil {
			return err
		}
		sealed, err := s.content.SealString(strconv.FormatInt(amountMinor, 10), debtPaymentAAD(paymentID))
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx,
			`UPDATE split_debt_payments SET amount_sealed = $2, amount_key_id = $3 WHERE id = $1`,
			paymentID, sealed.Ciphertext, sealed.KeyID,
		)
		return err
	})
	if err != nil {
		return Debt{}, err
	}

	// Return the debt as it now stands.
	debts, err := s.debtsByID(ctx, authorID, debtID)
	if err != nil {
		return Debt{}, err
	}
	return debts, nil
}

func (s *Service) paymentsOfTx(ctx context.Context, tx pgx.Tx, debtID int64) ([]DebtPayment, int64, error) {
	rows, err := tx.Query(ctx,
		`SELECT id, author_id, amount_sealed, amount_key_id, paid_at
		   FROM split_debt_payments WHERE debt_id = $1 ORDER BY paid_at, id`, debtID)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	out := make([]DebtPayment, 0, 4)
	total := int64(0)
	for rows.Next() {
		var payment DebtPayment
		var sealed []byte
		var keyID string
		if err := rows.Scan(&payment.ID, &payment.AuthorID, &sealed, &keyID, &payment.PaidAt); err != nil {
			return nil, 0, err
		}
		raw, err := s.content.OpenString(secure.Sealed{KeyID: keyID, Ciphertext: sealed}, debtPaymentAAD(payment.ID))
		if err != nil {
			return nil, 0, err
		}
		amount, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return nil, 0, err
		}
		payment.AmountMinor = amount
		total += amount
		out = append(out, payment)
	}
	return out, total, rows.Err()
}

// debtsByID returns one debt the viewer is allowed to see.
func (s *Service) debtsByID(ctx context.Context, viewerID, debtID int64) (Debt, error) {
	var debt Debt
	var amountSealed []byte
	var amountKey string
	var noteSealed []byte
	var noteKey *string

	err := s.pool.QueryRow(ctx,
		`SELECT d.id, d.group_id, d.author_id, d.creditor_id, d.debtor_id,
		        d.amount_sealed, d.amount_key_id, d.note_sealed, d.note_key_id,
		        d.source_expense_id, d.created_at
		   FROM split_debts d
		   JOIN split_members m ON m.group_id = d.group_id
		  WHERE d.id = $1 AND m.user_id = $2 AND m.state = 'active'`,
		debtID, viewerID,
	).Scan(&debt.ID, &debt.GroupID, &debt.AuthorID, &debt.CreditorID, &debt.DebtorID,
		&amountSealed, &amountKey, &noteSealed, &noteKey, &debt.SourceExpenseID, &debt.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Debt{}, httpapi.NotFound("Долг не найден")
		}
		return Debt{}, err
	}

	amountRaw, err := s.content.OpenString(secure.Sealed{KeyID: amountKey, Ciphertext: amountSealed}, debtAmountAAD(debt.ID))
	if err != nil {
		return Debt{}, err
	}
	amount, err := strconv.ParseInt(amountRaw, 10, 64)
	if err != nil {
		return Debt{}, err
	}
	debt.AmountMinor = amount

	if len(noteSealed) > 0 && noteKey != nil {
		note, err := s.content.OpenString(secure.Sealed{KeyID: *noteKey, Ciphertext: noteSealed}, debtNoteAAD(debt.ID))
		if err != nil {
			return Debt{}, err
		}
		debt.Note = note
	}

	payments, paid, err := s.paymentsOf(ctx, debt.ID)
	if err != nil {
		return Debt{}, err
	}
	debt.Payments = payments
	debt.PaidMinor = paid
	debt.Outstanding = amount - paid
	return debt, nil
}

// ManualDebtTotal is the sum of manual debts per person, used by the balance.
//
// Only rows without a source expense count: an expense-backed debt is already in
// the expense totals, and adding it again would double the money.
func (s *Service) ManualDebtTotals(ctx context.Context, groupID int64) (map[int64]int64, map[int64]int64, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, creditor_id, debtor_id, amount_sealed, amount_key_id
		   FROM split_debts WHERE group_id = $1 AND source_expense_id IS NULL`, groupID)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	owed := map[int64]int64{}
	owe := map[int64]int64{}
	for rows.Next() {
		var id, creditorID, debtorID int64
		var sealed []byte
		var keyID string
		if err := rows.Scan(&id, &creditorID, &debtorID, &sealed, &keyID); err != nil {
			return nil, nil, err
		}
		raw, err := s.content.OpenString(secure.Sealed{KeyID: keyID, Ciphertext: sealed}, debtAmountAAD(id))
		if err != nil {
			return nil, nil, err
		}
		amount, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return nil, nil, err
		}
		// Only the outstanding part counts: a repaid debt is settled money.
		_, paid, err := s.paymentsOf(ctx, id)
		if err != nil {
			return nil, nil, err
		}
		outstanding := amount - paid
		if outstanding <= 0 {
			continue
		}
		owed[creditorID] += outstanding
		owe[debtorID] += outstanding
	}
	return owed, owe, rows.Err()
}

func debtAmountAAD(debtID int64) string {
	return "split_debt_amount:" + strconv.FormatInt(debtID, 10)
}

func debtNoteAAD(debtID int64) string {
	return "split_debt_note:" + strconv.FormatInt(debtID, 10)
}

func debtPaymentAAD(paymentID int64) string {
	return "split_debt_payment:" + strconv.FormatInt(paymentID, 10)
}

// debtTime is used by tests to keep timestamps comparable.
var _ = time.Time{}
