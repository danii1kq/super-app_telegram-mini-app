package split

import (
	"context"
	"strconv"
	"strings"

	"github.com/danii1kq/srunal/internal/platform/httpapi"
	"github.com/danii1kq/srunal/internal/platform/ocr"
	"github.com/jackc/pgx/v5"
)

func normalizeReceiptItems(items []ReceiptItem) error {
	if len(items) == 0 || len(items) > ocr.MaxItems {
		return httpapi.BadRequest("invalid_items", "Добавьте от 1 до 200 позиций")
	}
	var total int64
	ids := map[int64]bool{}
	for index := range items {
		item := &items[index]
		item.Position = index
		item.Name = strings.TrimSpace(item.Name)
		if item.ID < 0 || item.ID > 0 && ids[item.ID] {
			return httpapi.BadRequest("invalid_items", "Повторный ID позиции")
		}
		ids[item.ID] = true
		if len([]rune(item.Name)) > 200 || item.Quantity < 1 || item.Quantity > ocr.MaxQuantity || item.AmountMinor < 0 || item.AmountMinor > ocr.MaxMoneyMinor || total > ocr.MaxMoneyMinor-item.AmountMinor {
			return httpapi.Validation(map[string]string{"items": "Проверьте название, количество и сумму позиции"}, "Проверьте чек")
		}
		total += item.AmountMinor
		if item.AllocationMode == "" {
			item.AllocationMode = "units"
			var units int
			for _, n := range item.Units {
				units += n
			}
			if units > item.Quantity {
				item.AllocationMode = "shared"
			}
		}
		if item.AllocationMode != "units" && item.AllocationMode != "shared" {
			return httpapi.BadRequest("invalid_mode", "Некорректный режим распределения")
		}
		if p := item.ParentPosition; p != nil {
			if *p < 0 || *p >= len(items) || *p == index || items[*p].ParentPosition != nil {
				return httpapi.BadRequest("invalid_combo", "Комбо должно ссылаться на основную позицию этого чека")
			}
			item.AllocationMode = items[*p].AllocationMode
			if item.AllocationMode == "" {
				item.AllocationMode = "units"
			}
			item.Units = items[*p].Units
		}
		if len(item.Units) > 100 {
			return httpapi.BadRequest("invalid_units", "Не больше 100 участников в позиции")
		}
		units := 0
		for id, n := range item.Units {
			if id <= 0 || n < 1 || n > 999 || item.AllocationMode == "shared" && n != 1 {
				return httpapi.Validation(map[string]string{"items": "Некорректные доли позиции"}, "Проверьте чек")
			}
			units += n
		}
		if item.ParentPosition == nil && item.AllocationMode == "units" && units > item.Quantity {
			return httpapi.Conflict("item_taken", "Единиц больше, чем в позиции")
		}
	}
	// Parents may follow their children in the displayed order.
	for i := range items {
		if p := items[i].ParentPosition; p != nil {
			items[i].Units = items[*p].Units
			items[i].AllocationMode = items[*p].AllocationMode
		}
	}
	return nil
}

// Stable IDs keep another participant's selection attached to the same item.
func (s *Service) UpdateReceipt(ctx context.Context, authorID, receiptID int64, total, discount, service int64, items []ReceiptItem, expected ...int) (Receipt, error) {
	if total < 0 || total > ocr.MaxMoneyMinor || discount < 0 || discount > ocr.MaxMoneyMinor || service < 0 || service > ocr.MaxMoneyMinor {
		return Receipt{}, httpapi.BadRequest("invalid_amount", "Проверьте суммы чека")
	}
	if err := normalizeReceiptItems(items); err != nil {
		return Receipt{}, err
	}
	err := s.pool.InTx(ctx, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `SELECT id FROM split_receipts WHERE id=$1 FOR UPDATE`, receiptID); err != nil {
			return err
		}
		receipt, err := s.receipt(ctx, tx, authorID, receiptID)
		if err != nil {
			return err
		}
		if receipt.AuthorID != authorID {
			return httpapi.Forbidden("Чек создал другой участник")
		}
		if receipt.Status != "draft" && receipt.Status != "collecting" {
			return httpapi.Conflict("receipt_locked", "Сначала верните чек к редактированию")
		}
		if len(expected) > 0 && receipt.Version != expected[0] {
			return httpapi.Conflict("version_conflict", "Чек изменился. Обновите его и повторите действие")
		}
		existing := map[int64]bool{}
		for _, i := range receipt.Items {
			existing[i.ID] = true
		}
		kept := []int64{}
		for _, i := range items {
			if i.ID > 0 {
				if !existing[i.ID] {
					return httpapi.BadRequest("invalid_item", "Позиция из другого чека")
				}
				kept = append(kept, i.ID)
			}
			for userID := range i.Units {
				var member bool
				if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM split_members WHERE group_id=$1 AND user_id=$2 AND state='active')`, receipt.GroupID, userID).Scan(&member); err != nil {
					return err
				}
				if !member {
					return httpapi.BadRequest("invalid_member", "Участник не входит в группу")
				}
			}
		}
		if _, err = tx.Exec(ctx, `DELETE FROM split_receipt_items WHERE receipt_id=$1 AND NOT(id=ANY($2::bigint[]))`, receiptID, kept); err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `UPDATE split_receipt_items SET position=-position-1 WHERE receipt_id=$1`, receiptID); err != nil {
			return err
		}
		for index, item := range items {
			id := item.ID
			if id == 0 {
				if err = tx.QueryRow(ctx, `INSERT INTO split_receipt_items(receipt_id,position,name_sealed,name_key_id,quantity,amount_sealed,amount_key_id) VALUES($1,$2,''::bytea,'pending',$3,''::bytea,'pending') RETURNING id`, receiptID, index, item.Quantity).Scan(&id); err != nil {
					return err
				}
			}
			nameSeal, err := s.content.SealString(item.Name, receiptItemNameAAD(id))
			if err != nil {
				return err
			}
			amountSeal, err := s.content.SealString(strconv.FormatInt(item.AmountMinor, 10), receiptItemAmountAAD(id))
			if err != nil {
				return err
			}
			if _, err = tx.Exec(ctx, `UPDATE split_receipt_items SET position=$2,name_sealed=$3,name_key_id=$4,quantity=$5,amount_sealed=$6,amount_key_id=$7,confidence=$8,allocation_mode=$9,parent_position=$10 WHERE id=$1`, id, index, nameSeal.Ciphertext, nameSeal.KeyID, item.Quantity, amountSeal.Ciphertext, amountSeal.KeyID, confidenceOf(item.Unclear), item.AllocationMode, item.ParentPosition); err != nil {
				return err
			}
			if _, err = tx.Exec(ctx, `DELETE FROM split_receipt_item_units WHERE item_id=$1`, id); err != nil {
				return err
			}
			for userID, n := range item.Units {
				if _, err = tx.Exec(ctx, `INSERT INTO split_receipt_item_units(item_id,user_id,units) VALUES($1,$2,$3)`, id, userID, n); err != nil {
					return err
				}
			}
		}
		totalSeal, totalKey, err := s.sealOptional(total, receiptTotalAAD(receiptID))
		if err != nil {
			return err
		}
		discountSeal, discountKey, err := s.sealOptional(discount, receiptDiscountAAD(receiptID))
		if err != nil {
			return err
		}
		serviceSeal, serviceKey, err := s.sealOptional(service, receiptServiceAAD(receiptID))
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `UPDATE split_receipts SET total_sealed=$2,total_key_id=$3,discount_sealed=$4,discount_key_id=$5,service_sealed=$6,service_key_id=$7,version=version+1,updated_at=now() WHERE id=$1`, receiptID, totalSeal, totalKey, discountSeal, discountKey, serviceSeal, serviceKey)
		return err
	})
	if err != nil {
		return Receipt{}, err
	}
	return s.Receipt(ctx, authorID, receiptID)
}
