package split

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strconv"
	"time"

	"github.com/danii1kq/srunal/internal/platform/httpapi"
	"github.com/danii1kq/srunal/internal/platform/secure"
	"github.com/jackc/pgx/v5"
)

type obligation struct {
	DebtID, ExpenseID, AuthorID, CreditorID, DebtorID, Amount, Remaining int64
	CreatedAt                                                            time.Time
	Pending                                                              bool
}
type SettlementPair struct {
	DebtorID      string `json:"debtor_id"`
	CreditorID    string `json:"creditor_id"`
	TransferMinor string `json:"transfer_minor"`
	OffsetMinor   string `json:"offset_minor"`
	GroupVersion  int64  `json:"group_version"`
	Pending       bool   `json:"pending"`
}
type Settlement struct {
	ID        string         `json:"id"`
	AuthorID  string         `json:"author_id"`
	UserA     string         `json:"user_a"`
	UserB     string         `json:"user_b"`
	Status    string         `json:"status"`
	AAccepted bool           `json:"a_accepted"`
	BAccepted bool           `json:"b_accepted"`
	Pair      SettlementPair `json:"pair"`
}
type settlementSnapshot struct {
	Pair        SettlementPair
	Obligations []obligation
}

func (s *Service) obligations(ctx context.Context, q receiptQuerier, groupID int64) ([]obligation, error) {
	out := []obligation{}
	rows, err := q.Query(ctx, `SELECT e.id,e.author_id,e.payer_id,sh.user_id,sh.amount_sealed,sh.key_id,e.created_at,coalesce(d.id,0)
 FROM split_expenses e JOIN split_shares sh ON sh.expense_id=e.id LEFT JOIN split_debts d ON d.source_expense_id=e.id AND d.debtor_id=sh.user_id
 WHERE e.group_id=$1 AND e.deleted_at IS NULL AND e.payer_id<>sh.user_id`, groupID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var o obligation
		var sealed []byte
		var key string
		if err = rows.Scan(&o.ExpenseID, &o.AuthorID, &o.CreditorID, &o.DebtorID, &sealed, &key, &o.CreatedAt, &o.DebtID); err != nil {
			rows.Close()
			return nil, err
		}
		raw, err := s.content.OpenString(secure.Sealed{Ciphertext: sealed, KeyID: key}, shareAAD(o.ExpenseID, o.DebtorID))
		if err != nil {
			rows.Close()
			return nil, err
		}
		o.Amount, err = strconv.ParseInt(raw, 10, 64)
		if err != nil {
			rows.Close()
			return nil, err
		}
		o.Remaining = o.Amount
		out = append(out, o)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	rows, err = q.Query(ctx, `SELECT id,author_id,creditor_id,debtor_id,amount_sealed,amount_key_id,created_at FROM split_debts WHERE group_id=$1 AND source_expense_id IS NULL`, groupID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var o obligation
		var sealed []byte
		var key string
		if err = rows.Scan(&o.DebtID, &o.AuthorID, &o.CreditorID, &o.DebtorID, &sealed, &key, &o.CreatedAt); err != nil {
			rows.Close()
			return nil, err
		}
		raw, err := s.content.OpenString(secure.Sealed{Ciphertext: sealed, KeyID: key}, debtAmountAAD(o.DebtID))
		if err != nil {
			rows.Close()
			return nil, err
		}
		o.Amount, err = strconv.ParseInt(raw, 10, 64)
		if err != nil {
			rows.Close()
			return nil, err
		}
		o.Remaining = o.Amount
		out = append(out, o)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	rows, err = q.Query(ctx, `SELECT p.id,p.debt_id,p.amount_sealed,p.amount_key_id,p.status FROM split_debt_payments p JOIN split_debts d ON d.id=p.debt_id WHERE d.group_id=$1 AND p.status IN ('confirmed','pending')`, groupID)
	if err != nil {
		return nil, err
	}
	paid := map[int64]int64{}
	pending := map[int64]bool{}
	for rows.Next() {
		var id, debtID int64
		var sealed []byte
		var key, status string
		if err = rows.Scan(&id, &debtID, &sealed, &key, &status); err != nil {
			rows.Close()
			return nil, err
		}
		if status == "pending" {
			pending[debtID] = true
			continue
		}
		raw, err := s.content.OpenString(secure.Sealed{Ciphertext: sealed, KeyID: key}, debtPaymentAAD(id))
		if err != nil {
			rows.Close()
			return nil, err
		}
		amount, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			rows.Close()
			return nil, err
		}
		paid[debtID] += amount
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	result := []obligation{}
	for _, o := range out {
		o.Remaining -= paid[o.DebtID]
		o.Pending = pending[o.DebtID]
		if o.Remaining < 0 {
			return nil, errors.New("split: negative obligation remainder")
		}
		if o.Remaining > 0 {
			result = append(result, o)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if !result[i].CreatedAt.Equal(result[j].CreatedAt) {
			return result[i].CreatedAt.Before(result[j].CreatedAt)
		}
		if result[i].ExpenseID != result[j].ExpenseID {
			return result[i].ExpenseID < result[j].ExpenseID
		}
		return result[i].DebtID < result[j].DebtID
	})
	return result, nil
}

func pairFor(obligations []obligation, a, b, version int64) (SettlementPair, []obligation) {
	selected := []obligation{}
	var ab, ba int64
	pending := false
	for _, o := range obligations {
		if o.DebtorID == a && o.CreditorID == b {
			ab += o.Remaining
		} else if o.DebtorID == b && o.CreditorID == a {
			ba += o.Remaining
		} else {
			continue
		}
		selected = append(selected, o)
		pending = pending || o.Pending
	}
	debtor, creditor := a, b
	transfer := ab - ba
	if transfer < 0 {
		debtor, creditor = b, a
		transfer = -transfer
	}
	offset := min(ab, ba)
	return SettlementPair{DebtorID: strconv.FormatInt(debtor, 10), CreditorID: strconv.FormatInt(creditor, 10), TransferMinor: strconv.FormatInt(transfer, 10), OffsetMinor: strconv.FormatInt(offset, 10), GroupVersion: version, Pending: pending}, selected
}

func (s *Service) SettlementPlan(ctx context.Context, userID, groupID int64) ([]SettlementPair, []Settlement, error) {
	if err := s.requireMember(ctx, userID, groupID); err != nil {
		return nil, nil, err
	}
	pairs := []SettlementPair{}
	proposals := []Settlement{}
	err := s.pool.InTx(ctx, func(tx pgx.Tx) error {
		if err := lockGroup(ctx, tx, groupID); err != nil {
			return err
		}
		var version int64
		if err := tx.QueryRow(ctx, `SELECT financial_version FROM split_groups WHERE id=$1`, groupID).Scan(&version); err != nil {
			return err
		}
		all, err := s.obligations(ctx, tx, groupID)
		if err != nil {
			return err
		}
		seen := map[[2]int64]bool{}
		for _, o := range all {
			a, b := min(o.CreditorID, o.DebtorID), max(o.CreditorID, o.DebtorID)
			key := [2]int64{a, b}
			if seen[key] {
				continue
			}
			seen[key] = true
			p, _ := pairFor(all, a, b, version)
			pairs = append(pairs, p)
		}
		if _, err = tx.Exec(ctx, `UPDATE split_settlements SET status='stale' WHERE group_id=$1 AND group_version<>$2 AND status IN ('proposed','agreed','reported')`, groupID, version); err != nil {
			return err
		}
		rows, err := tx.Query(ctx, `SELECT id FROM split_settlements WHERE group_id=$1 ORDER BY id DESC LIMIT 100`, groupID)
		if err != nil {
			return err
		}
		ids := []int64{}
		for rows.Next() {
			var id int64
			if err = rows.Scan(&id); err != nil {
				rows.Close()
				return err
			}
			ids = append(ids, id)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return err
		}
		for _, id := range ids {
			p, _, err := s.settlement(ctx, tx, id)
			if err != nil {
				return err
			}
			proposals = append(proposals, p)
		}
		return nil
	})
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].DebtorID != pairs[j].DebtorID {
			return pairs[i].DebtorID < pairs[j].DebtorID
		}
		return pairs[i].CreditorID < pairs[j].CreditorID
	})
	return pairs, proposals, err
}

func (s *Service) settlement(ctx context.Context, q receiptQuerier, id int64) (Settlement, settlementSnapshot, error) {
	var p Settlement
	var snapshot settlementSnapshot
	var raw []byte
	var key string
	var author, a, b int64
	err := q.QueryRow(ctx, `SELECT author_id,user_a,user_b,status,a_accepted,b_accepted,snapshot_sealed,snapshot_key_id FROM split_settlements WHERE id=$1`, id).Scan(&author, &a, &b, &p.Status, &p.AAccepted, &p.BAccepted, &raw, &key)
	if err != nil {
		return p, snapshot, err
	}
	plain, err := s.content.Open(secure.Sealed{Ciphertext: raw, KeyID: key}, "split_settlement:"+strconv.FormatInt(id, 10))
	if err != nil {
		return p, snapshot, err
	}
	if err = json.Unmarshal(plain, &snapshot); err != nil {
		return p, snapshot, err
	}
	p.ID = strconv.FormatInt(id, 10)
	p.AuthorID = strconv.FormatInt(author, 10)
	p.UserA = strconv.FormatInt(a, 10)
	p.UserB = strconv.FormatInt(b, 10)
	p.Pair = snapshot.Pair
	return p, snapshot, nil
}

func (s *Service) ProposeSettlement(ctx context.Context, userID, groupID, otherID, version int64, requestID string) (Settlement, error) {
	if otherID == userID || otherID <= 0 || requestID == "" || len(requestID) > 128 {
		return Settlement{}, httpapi.BadRequest("invalid_settlement", "Проверьте участников расчёта")
	}
	if err := s.requireMember(ctx, userID, groupID); err != nil {
		return Settlement{}, err
	}
	if err := s.requireMember(ctx, otherID, groupID); err != nil {
		return Settlement{}, err
	}
	var id int64
	err := s.pool.InTx(ctx, func(tx pgx.Tx) error {
		if err := lockGroup(ctx, tx, groupID); err != nil {
			return err
		}
		a, b := min(userID, otherID), max(userID, otherID)
		err := tx.QueryRow(ctx, `SELECT id FROM split_settlements WHERE author_id=$1 AND client_request_id=$2 AND group_id=$3 AND user_a=$4 AND user_b=$5`, userID, requestID, groupID, a, b).Scan(&id)
		if err == nil {
			return nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		var current int64
		if err = tx.QueryRow(ctx, `SELECT financial_version FROM split_groups WHERE id=$1`, groupID).Scan(&current); err != nil {
			return err
		}
		if current != version {
			return httpapi.Conflict("version_conflict", "Долги изменились, обновите расчёт")
		}
		if _, err = tx.Exec(ctx, `UPDATE split_settlements SET status='stale' WHERE group_id=$1 AND user_a=$2 AND user_b=$3 AND group_version<>$4 AND status IN ('proposed','agreed','reported')`, groupID, a, b, current); err != nil {
			return err
		}
		err = tx.QueryRow(ctx, `SELECT id FROM split_settlements WHERE group_id=$1 AND user_a=$2 AND user_b=$3 AND status IN ('proposed','agreed','reported')`, groupID, a, b).Scan(&id)
		if err == nil {
			return nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		all, err := s.obligations(ctx, tx, groupID)
		if err != nil {
			return err
		}
		pair, selected := pairFor(all, a, b, version)
		if len(selected) == 0 || pair.Pending {
			return httpapi.Conflict("settlement_unavailable", "Нет доступного остатка или ожидается подтверждение перевода")
		}
		if err = tx.QueryRow(ctx, `INSERT INTO split_settlements(group_id,author_id,user_a,user_b,group_version,snapshot_sealed,snapshot_key_id,a_accepted,b_accepted,client_request_id)VALUES($1,$2,$3,$4,$5,''::bytea,'pending',$6,$7,$8)RETURNING id`, groupID, userID, a, b, version, userID == a, userID == b, requestID).Scan(&id); err != nil {
			return err
		}
		plain, err := json.Marshal(settlementSnapshot{Pair: pair, Obligations: selected})
		if err != nil {
			return err
		}
		sealed, err := s.content.Seal(plain, "split_settlement:"+strconv.FormatInt(id, 10))
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `UPDATE split_settlements SET snapshot_sealed=$2,snapshot_key_id=$3 WHERE id=$1`, id, sealed.Ciphertext, sealed.KeyID)
		return err
	})
	if err != nil {
		return Settlement{}, err
	}
	p, _, err := s.settlement(ctx, s.pool, id)
	return p, err
}

func (s *Service) RespondSettlement(ctx context.Context, userID, id int64, action string) (Settlement, error) {
	err := s.pool.InTx(ctx, func(tx pgx.Tx) error {
		var groupID int64
		if err := tx.QueryRow(ctx, `SELECT group_id FROM split_settlements WHERE id=$1`, id).Scan(&groupID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return httpapi.NotFound("Расчёт не найден")
			}
			return err
		}
		if err := lockGroup(ctx, tx, groupID); err != nil {
			return err
		}
		var member bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM split_members WHERE group_id=$1 AND user_id=$2 AND state='active')`, groupID, userID).Scan(&member); err != nil {
			return err
		}
		if !member {
			return httpapi.NotFound("Расчёт не найден")
		}
		p, snapshot, err := s.settlement(ctx, tx, id)
		if err != nil {
			return err
		}
		uid := strconv.FormatInt(userID, 10)
		if uid != p.UserA && uid != p.UserB {
			return httpapi.Forbidden("Расчёт подтверждают две его стороны")
		}
		if p.Status == "confirmed" || p.Status == "rejected" || p.Status == "stale" {
			return nil
		}
		var version int64
		if err = tx.QueryRow(ctx, `SELECT financial_version FROM split_groups WHERE id=$1`, groupID).Scan(&version); err != nil {
			return err
		}
		if version != snapshot.Pair.GroupVersion {
			_, err = tx.Exec(ctx, `UPDATE split_settlements SET status='stale' WHERE id=$1`, id)
			return err
		}
		status := p.Status
		switch action {
		case "reject":
			status = "rejected"
		case "agree":
			if uid == p.UserA {
				p.AAccepted = true
			} else {
				p.BAccepted = true
			}
			if p.AAccepted && p.BAccepted && status == "proposed" {
				status = "agreed"
			}
			if status == "agreed" && p.Pair.TransferMinor == "0" {
				if err = s.applySettlement(ctx, tx, groupID, userID, id, snapshot); err != nil {
					return err
				}
				status = "confirmed"
			}
		case "report":
			if uid != p.Pair.DebtorID || status != "agreed" {
				return httpapi.Conflict("invalid_settlement_state", "Сначала согласуйте расчёт обеими сторонами")
			}
			status = "reported"
		case "confirm":
			if uid != p.Pair.CreditorID || (status != "reported" && status != "agreed") {
				return httpapi.Forbidden("Получение подтверждает получатель согласованного перевода")
			}
			if err = s.applySettlement(ctx, tx, groupID, userID, id, snapshot); err != nil {
				return err
			}
			status = "confirmed"
		default:
			return httpapi.BadRequest("invalid_action", "Некорректное действие")
		}
		_, err = tx.Exec(ctx, `UPDATE split_settlements SET status=$2,a_accepted=$3,b_accepted=$4 WHERE id=$1`, id, status, p.AAccepted, p.BAccepted)
		return err
	})
	if err != nil {
		return Settlement{}, err
	}
	p, _, err := s.settlement(ctx, s.pool, id)
	return p, err
}

func (s *Service) applySettlement(ctx context.Context, tx pgx.Tx, groupID, actorID, settlementID int64, snapshot settlementSnapshot) error {
	offset, err := strconv.ParseInt(snapshot.Pair.OffsetMinor, 10, 64)
	if err != nil {
		return err
	}
	transfer, err := strconv.ParseInt(snapshot.Pair.TransferMinor, 10, 64)
	if err != nil {
		return err
	}
	debtor, err := strconv.ParseInt(snapshot.Pair.DebtorID, 10, 64)
	if err != nil {
		return err
	}
	remainingOffset := map[int64]int64{}
	for _, o := range snapshot.Obligations {
		remainingOffset[o.DebtorID] = offset
	}
	remainingTransfer := transfer
	for index, o := range snapshot.Obligations {
		usedOffset := min(o.Remaining, remainingOffset[o.DebtorID])
		remainingOffset[o.DebtorID] -= usedOffset
		cash := int64(0)
		if o.DebtorID == debtor {
			cash = min(o.Remaining-usedOffset, remainingTransfer)
			remainingTransfer -= cash
		}
		if usedOffset+cash == 0 {
			continue
		}
		debtID := o.DebtID
		if debtID == 0 {
			if err = tx.QueryRow(ctx, `INSERT INTO split_debts(group_id,author_id,creditor_id,debtor_id,amount_sealed,amount_key_id,source_expense_id,client_request_id)VALUES($1,$2,$3,$4,''::bytea,'pending',$5,$6)RETURNING id`, groupID, actorID, o.CreditorID, o.DebtorID, o.ExpenseID, "settlement-source:"+strconv.FormatInt(o.ExpenseID, 10)+":"+strconv.FormatInt(o.DebtorID, 10)).Scan(&debtID); err != nil {
				return err
			}
			sealed, err := s.content.SealString(strconv.FormatInt(o.Amount, 10), debtAmountAAD(debtID))
			if err != nil {
				return err
			}
			if _, err = tx.Exec(ctx, `UPDATE split_debts SET amount_sealed=$2,amount_key_id=$3 WHERE id=$1`, debtID, sealed.Ciphertext, sealed.KeyID); err != nil {
				return err
			}
		}
		d, err := s.lockedDebt(ctx, tx, actorID, debtID)
		if err != nil {
			return err
		}
		key := "settlement:" + strconv.FormatInt(settlementID, 10) + ":" + strconv.Itoa(index)
		if usedOffset > 0 {
			if _, err = s.addPaymentTx(ctx, tx, d, actorID, usedOffset, key+":offset", "confirmed", "offset", true); err != nil {
				return err
			}
		}
		if cash > 0 {
			if _, err = s.addPaymentTx(ctx, tx, d, actorID, cash, key+":cash", "confirmed", "transfer", true); err != nil {
				return err
			}
		}
	}
	if remainingTransfer != 0 {
		return errors.New("split: settlement did not allocate all money")
	}
	for _, n := range remainingOffset {
		if n != 0 {
			return errors.New("split: offset did not allocate all money")
		}
	}
	return nil
}
