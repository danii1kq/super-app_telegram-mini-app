package split

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/danii1kq/srunal/internal/platform/httpapi"
	"github.com/danii1kq/srunal/internal/platform/secure"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// SaveReceipt commits checked positions once, even when the response is lost.
func (s *Service) SaveReceipt(ctx context.Context, authorID, receiptID, payerID int64, clientRequestID, description, category string) (Expense, error) {
	if clientRequestID == "" || len(clientRequestID) > 128 {
		return Expense{}, httpapi.BadRequest("invalid_request_id", "Передайте client_request_id до 128 символов")
	}
	description = strings.TrimSpace(description)
	if description == "" {
		description = "Чек"
	}
	if len([]rune(description)) > 200 {
		return Expense{}, httpapi.BadRequest("description_length", "Название до 200 символов")
	}
	category = normalizeCategory(category)
	var expenseID, groupID int64
	err := s.pool.InTx(ctx, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, `SELECT group_id FROM split_receipts WHERE id=$1 FOR UPDATE`, receiptID).Scan(&groupID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return httpapi.NotFound("Чек не найден")
			}
			return err
		}
		receipt, err := s.receipt(ctx, tx, authorID, receiptID)
		if err != nil {
			return err
		}
		if receipt.AuthorID != authorID {
			return httpapi.Forbidden("Чек создал другой участник")
		}
		// Serialize the sender's request keys across receipts without nested pool queries.
		if _, err = tx.Exec(ctx, `SELECT id FROM users WHERE id=$1 FOR UPDATE`, authorID); err != nil {
			return err
		}
		var existingGroup, existingPayer int64
		var sealed []byte
		var keyID, existingCategory string
		err = tx.QueryRow(ctx, `SELECT id,group_id,payer_id,description_sealed,key_id,coalesce(category,'other') FROM split_expenses WHERE author_id=$1 AND client_request_id=$2 AND deleted_at IS NULL`, authorID, clientRequestID).Scan(&expenseID, &existingGroup, &existingPayer, &sealed, &keyID, &existingCategory)
		if err == nil {
			existingDescription, err := s.content.OpenString(secure.Sealed{KeyID: keyID, Ciphertext: sealed}, expenseDescriptionAAD(expenseID, authorID))
			if err != nil {
				return err
			}
			if receipt.SavedExpenseID == nil || *receipt.SavedExpenseID != expenseID || existingGroup != groupID || existingPayer != payerID || existingDescription != description || existingCategory != category {
				return httpapi.Conflict("idempotency_conflict", "Этот ключ уже использован для другого расхода")
			}
			return nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		if receipt.Status != "draft" {
			return httpapi.Conflict("receipt_saved", "Чек уже сохранён как расход")
		}
		if len(receipt.Items) == 0 {
			return httpapi.Validation(map[string]string{"items": "Добавьте позиции"}, "Проверьте чек")
		}
		participants := map[int64]bool{authorID: true, payerID: true}
		for _, item := range receipt.Items {
			if item.Unclear || strings.TrimSpace(item.Name) == "" || item.AmountMinor <= 0 {
				return httpapi.Validation(map[string]string{"items": "Проверьте название и цену каждой позиции"}, "Подтвердите распознанные позиции")
			}
			for id := range item.Units {
				participants[id] = true
			}
		}
		ids := make([]int64, 0, len(participants))
		for id := range participants {
			ids = append(ids, id)
		}
		rows, err := tx.Query(ctx, `SELECT user_id FROM split_members WHERE group_id=$1 AND user_id=ANY($2::bigint[]) AND state='active' FOR SHARE`, groupID, ids)
		if err != nil {
			return err
		}
		count := 0
		for rows.Next() {
			count++
		}
		readErr := rows.Err()
		rows.Close()
		if readErr != nil {
			return readErr
		}
		if count != len(ids) {
			return httpapi.Validation(map[string]string{"payer_id": "Все участники должны состоять в группе"}, "Проверьте участников")
		}
		allocation := Allocate(receipt, payerID)
		if len(allocation.Unallocated) > 0 {
			return httpapi.Validation(map[string]string{"items": "Есть позиции, которые никто не взял"}, "Распределите все позиции")
		}
		if allocation.Difference != 0 || allocation.SumShares <= 0 || !CheckShares(receipt.TotalMinor, allocation.PerPerson) {
			return httpapi.Validation(map[string]string{"total_minor": "Сумма долей не совпадает с итогом чека"}, "Проверьте суммы")
		}
		err = tx.QueryRow(ctx, `INSERT INTO split_expenses(group_id,author_id,payer_id,description_sealed,amount_sealed,key_id,client_request_id,category) VALUES($1,$2,$3,''::bytea,''::bytea,'pending',$4,$5) RETURNING id`, groupID, authorID, payerID, clientRequestID, category).Scan(&expenseID)
		if err != nil {
			var conflict *pgconn.PgError
			if errors.As(err, &conflict) && conflict.Code == "23505" {
				return httpapi.Conflict("idempotency_conflict", "Этот ключ уже использован для другого расхода")
			}
			return err
		}
		descriptionSealed, err := s.content.SealString(description, expenseDescriptionAAD(expenseID, authorID))
		if err != nil {
			return err
		}
		amountSealed, err := s.content.SealString(strconv.FormatInt(receipt.TotalMinor, 10), expenseAmountAAD(expenseID, authorID))
		if err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `UPDATE split_expenses SET description_sealed=$2,amount_sealed=$3,key_id=$4 WHERE id=$1`, expenseID, descriptionSealed.Ciphertext, amountSealed.Ciphertext, descriptionSealed.KeyID); err != nil {
			return err
		}
		for id, amount := range allocation.PerPerson {
			sealed, err := s.content.SealString(strconv.FormatInt(amount, 10), shareAAD(expenseID, id))
			if err != nil {
				return err
			}
			if _, err = tx.Exec(ctx, `INSERT INTO split_shares(expense_id,user_id,amount_sealed,key_id)VALUES($1,$2,$3,$4)`, expenseID, id, sealed.Ciphertext, sealed.KeyID); err != nil {
				return err
			}
		}
		_, err = tx.Exec(ctx, `UPDATE split_receipts SET status='saved',saved_expense_id=$2,image_expires_at=now()+$3::interval,updated_at=now() WHERE id=$1`, receiptID, expenseID, savedImageTTL)
		return err
	})
	if err != nil {
		return Expense{}, err
	}
	return s.Expense(ctx, authorID, groupID, expenseID)
}

func (s *Service) openSealedString(sealed []byte, keyID *string, aad string) (string, error) {
	if len(sealed) == 0 || keyID == nil {
		return "", nil
	}
	return s.content.OpenString(secure.Sealed{KeyID: *keyID, Ciphertext: sealed}, aad)
}
