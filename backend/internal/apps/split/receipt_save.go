package split

import (
	"context"
	"errors"
	"strconv"

	"github.com/jackc/pgx/v5"

	"github.com/danii1kq/srunal/internal/platform/httpapi"
	"github.com/danii1kq/srunal/internal/platform/secure"
)

// SaveReceipt commits a checked draft as one expense.
//
// The shares come from the item units, so a line that belongs to one person is
// charged to that person only. The expense is the single source of the money:
// recording a debt afterwards points at it and never adds the amount again.
func (s *Service) SaveReceipt(ctx context.Context, authorID, receiptID, payerID int64, clientRequestID, description, category string) (Expense, error) {
	if clientRequestID == "" {
		return Expense{}, httpapi.BadRequest("missing_request_id", "Не передан client_request_id")
	}

	receipt, err := s.Receipt(ctx, authorID, receiptID)
	if err != nil {
		return Expense{}, err
	}
	if receipt.AuthorID != authorID {
		return Expense{}, httpapi.Forbidden("Чек создал другой участник")
	}
	if receipt.Status != "draft" {
		return Expense{}, httpapi.Conflict("receipt_saved", "Чек уже сохранён как расход")
	}
	if err := s.requireMember(ctx, payerID, receipt.GroupID); err != nil {
		return Expense{}, httpapi.Validation(map[string]string{"payer_id": "Плательщик не входит в группу"}, "Проверьте поля")
	}

	allocation := Allocate(receipt, payerID)
	if len(allocation.Unallocated) > 0 {
		return Expense{}, httpapi.Validation(
			map[string]string{"items": "Есть позиции, которые никто не взял"},
			"Распределите все позиции",
		)
	}
	if allocation.Difference != 0 {
		return Expense{}, httpapi.Validation(
			map[string]string{"total_minor": "Сумма долей не совпадает с итогом чека"},
			"Проверьте суммы",
		)
	}
	if len(allocation.PerPerson) == 0 {
		return Expense{}, httpapi.Validation(map[string]string{"items": "Ни одна позиция не распределена"}, "Проверьте поля")
	}

	// The expense amount is the receipt total; the shares are what each person
	// owes. Both must agree, which Allocate has just checked.
	amountMinor := receipt.TotalMinor
	if amountMinor == 0 {
		amountMinor = allocation.SumShares
	}
	if !CheckShares(amountMinor, allocation.PerPerson) {
		return Expense{}, httpapi.Internal(errors.New("split: receipt shares do not add up"))
	}
	if description == "" {
		description = "Чек"
	}

	var expense Expense
	err = s.pool.InTx(ctx, func(tx pgx.Tx) error {
		// Idempotency: a retry returns the expense the first attempt created.
		var existingID int64
		err := tx.QueryRow(ctx,
			`SELECT id FROM split_expenses WHERE author_id = $1 AND client_request_id = $2`,
			authorID, clientRequestID,
		).Scan(&existingID)
		if err == nil {
			expense.ID = existingID
			return nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}

		if err := tx.QueryRow(ctx,
			`INSERT INTO split_expenses (group_id, author_id, payer_id, description_sealed, amount_sealed, key_id, client_request_id, category)
			 VALUES ($1, $2, $3, ''::bytea, ''::bytea, 'pending', $4, $5)
			 RETURNING id, spent_at, version`,
			receipt.GroupID, authorID, payerID, clientRequestID, normalizeCategory(category),
		).Scan(&expense.ID, &expense.SpentAt, &expense.Version); err != nil {
			return err
		}

		descriptionSealed, err := s.content.SealString(description, expenseDescriptionAAD(expense.ID, authorID))
		if err != nil {
			return err
		}
		amountSealed, err := s.content.SealString(strconv.FormatInt(amountMinor, 10), expenseAmountAAD(expense.ID, authorID))
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx,
			`UPDATE split_expenses SET description_sealed = $2, amount_sealed = $3, key_id = $4 WHERE id = $1`,
			expense.ID, descriptionSealed.Ciphertext, amountSealed.Ciphertext, descriptionSealed.KeyID,
		); err != nil {
			return err
		}

		for participant, share := range allocation.PerPerson {
			shareSealed, err := s.content.SealString(strconv.FormatInt(share, 10), shareAAD(expense.ID, participant))
			if err != nil {
				return err
			}
			if _, err := tx.Exec(ctx,
				`INSERT INTO split_shares (expense_id, user_id, amount_sealed, key_id) VALUES ($1, $2, $3, $4)`,
				expense.ID, participant, shareSealed.Ciphertext, shareSealed.KeyID,
			); err != nil {
				return err
			}
		}

		// The draft is now an expense: mark it so a second save is a conflict.
		if _, err := tx.Exec(ctx,
			`UPDATE split_receipts SET status = 'saved', saved_expense_id = $2, updated_at = now() WHERE id = $1`,
			receiptID, expense.ID,
		); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return Expense{}, err
	}

	expense.GroupID = receipt.GroupID
	expense.AuthorID = authorID
	expense.PayerID = payerID
	expense.Description = description
	expense.AmountMinor = amountMinor
	expense.Shares = allocation.PerPerson
	return expense, nil
}

// openSealedString is a small helper for reading a value that may be absent.
func (s *Service) openSealedString(sealed []byte, keyID *string, aad string) (string, error) {
	if len(sealed) == 0 || keyID == nil {
		return "", nil
	}
	return s.content.OpenString(secure.Sealed{KeyID: *keyID, Ciphertext: sealed}, aad)
}
