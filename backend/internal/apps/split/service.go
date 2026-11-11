// Package split owns SruSplit: expense groups, expenses and shares.
//
// Money is integer minor units everywhere. Amounts and descriptions are sealed;
// the balance is computed in Go after decrypting, never with SQL SUM over
// ciphertext.
package split

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/danii1kq/srunal/internal/platform/httpapi"
	"github.com/danii1kq/srunal/internal/platform/postgres"
	"github.com/danii1kq/srunal/internal/platform/secure"
)

type Service struct {
	pool    *postgres.Pool
	content *secure.Cipher
}

func NewService(pool *postgres.Pool, content *secure.Cipher) *Service {
	return &Service{pool: pool, content: content}
}

type Group struct {
	ID        int64
	Title     string
	OwnerID   int64
	CreatedAt time.Time
}

type Expense struct {
	ID          int64
	GroupID     int64
	AuthorID    int64
	PayerID     int64
	Description string
	AmountMinor int64
	SpentAt     time.Time
	Version     int
	Shares      map[int64]int64
}

type Balance struct {
	UserID       int64
	PaidMinor    int64
	ShareMinor   int64
	BalanceMinor int64
}

// ListGroups returns the groups the viewer belongs to.
func (s *Service) ListGroups(ctx context.Context, viewerID int64) ([]Group, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT g.id, g.title_sealed, g.key_id, g.owner_id, g.created_at
		   FROM split_groups g
		   JOIN split_members m ON m.group_id = g.id
		  WHERE m.user_id = $1 AND m.state = 'active' AND g.archived_at IS NULL
		  ORDER BY g.created_at DESC, g.id DESC`, viewerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]Group, 0, 8)
	for rows.Next() {
		var group Group
		var sealed []byte
		var keyID string
		if err := rows.Scan(&group.ID, &sealed, &keyID, &group.OwnerID, &group.CreatedAt); err != nil {
			return nil, err
		}
		title, err := s.content.OpenString(secure.Sealed{KeyID: keyID, Ciphertext: sealed}, groupTitleAAD(group.ID))
		if err != nil {
			return nil, err
		}
		group.Title = title
		out = append(out, group)
	}
	return out, rows.Err()
}

// CreateGroup creates a group with the owner as the first member.
func (s *Service) CreateGroup(ctx context.Context, ownerID int64, title string, members []int64) (Group, error) {
	if title == "" {
		return Group{}, httpapi.Validation(map[string]string{"title": "Название не может быть пустым"}, "Проверьте поля")
	}

	var group Group
	err := s.pool.InTx(ctx, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx,
			`INSERT INTO split_groups (owner_id, title_sealed, key_id) VALUES ($1, ''::bytea, 'pending')
			 RETURNING id, created_at`, ownerID,
		).Scan(&group.ID, &group.CreatedAt); err != nil {
			return err
		}

		sealed, err := s.content.SealString(title, groupTitleAAD(group.ID))
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx,
			`UPDATE split_groups SET title_sealed = $2, key_id = $3 WHERE id = $1`,
			group.ID, sealed.Ciphertext, sealed.KeyID,
		); err != nil {
			return err
		}

		if _, err := tx.Exec(ctx,
			`INSERT INTO split_members (group_id, user_id) VALUES ($1, $2)
			 ON CONFLICT DO NOTHING`, group.ID, ownerID,
		); err != nil {
			return err
		}
		for _, member := range members {
			if member == ownerID {
				continue
			}
			if _, err := tx.Exec(ctx,
				`INSERT INTO split_members (group_id, user_id) VALUES ($1, $2)
				 ON CONFLICT DO NOTHING`, group.ID, member,
			); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return Group{}, err
	}

	group.Title = title
	group.OwnerID = ownerID
	return group, nil
}

// AddExpense stores an expense with equal shares.
//
// The idempotency key is the author's client_request_id: a retry returns the
// stored expense instead of creating a second one.
func (s *Service) AddExpense(ctx context.Context, authorID, groupID, payerID int64, clientRequestID, description string, amountMinor int64, participants []int64) (Expense, error) {
	if clientRequestID == "" {
		return Expense{}, httpapi.BadRequest("missing_request_id", "Не передан client_request_id")
	}
	if amountMinor <= 0 {
		return Expense{}, httpapi.Validation(map[string]string{"amount_minor": "Сумма должна быть больше нуля"}, "Проверьте поля")
	}

	shares, err := Split(amountMinor, participants)
	if err != nil {
		return Expense{}, httpapi.Validation(map[string]string{"participants": "Выберите участников разделения"}, "Проверьте поля")
	}
	if !CheckShares(amountMinor, shares) {
		return Expense{}, httpapi.Internal(errors.New("split: shares do not add up"))
	}

	var expense Expense
	err = s.pool.InTx(ctx, func(tx pgx.Tx) error {
		// Everyone involved must be an active member of the group.
		for _, participant := range participants {
			var member bool
			if err := tx.QueryRow(ctx,
				`SELECT EXISTS (SELECT 1 FROM split_members
				                 WHERE group_id = $1 AND user_id = $2 AND state = 'active')`,
				groupID, participant,
			).Scan(&member); err != nil {
				return err
			}
			if !member {
				return httpapi.Forbidden("Участник не входит в группу расходов")
			}
		}
		var payerMember bool
		if err := tx.QueryRow(ctx,
			`SELECT EXISTS (SELECT 1 FROM split_members
			                 WHERE group_id = $1 AND user_id = $2 AND state = 'active')`,
			groupID, payerID,
		).Scan(&payerMember); err != nil {
			return err
		}
		if !payerMember {
			return httpapi.Forbidden("Плательщик не входит в группу расходов")
		}

		// Idempotency: the same author + request id returns the stored expense.
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
			`INSERT INTO split_expenses (group_id, author_id, payer_id, description_sealed, amount_sealed, key_id, client_request_id)
			 VALUES ($1, $2, $3, ''::bytea, ''::bytea, 'pending', $4)
			 RETURNING id, spent_at, version`,
			groupID, authorID, payerID, clientRequestID,
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

		for participant, share := range shares {
			shareSealed, err := s.content.SealString(strconv.FormatInt(share, 10), shareAAD(expense.ID, participant))
			if err != nil {
				return err
			}
			if _, err := tx.Exec(ctx,
				`INSERT INTO split_shares (expense_id, user_id, amount_sealed, key_id)
				 VALUES ($1, $2, $3, $4)`,
				expense.ID, participant, shareSealed.Ciphertext, shareSealed.KeyID,
			); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return Expense{}, err
	}

	expense.GroupID = groupID
	expense.AuthorID = authorID
	expense.PayerID = payerID
	expense.Description = description
	expense.AmountMinor = amountMinor
	expense.Shares = shares
	return expense, nil
}

// Expenses lists the group's expenses, newest first.
func (s *Service) Expenses(ctx context.Context, viewerID, groupID int64) ([]Expense, error) {
	if err := s.requireMember(ctx, viewerID, groupID); err != nil {
		return nil, err
	}

	rows, err := s.pool.Query(ctx,
		`SELECT id, author_id, payer_id, description_sealed, amount_sealed, key_id, spent_at, version
		   FROM split_expenses
		  WHERE group_id = $1 AND deleted_at IS NULL
		  ORDER BY spent_at DESC, id DESC
		  LIMIT 200`, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]Expense, 0, 16)
	for rows.Next() {
		var expense Expense
		var descriptionSealed, amountSealed []byte
		var keyID string
		if err := rows.Scan(&expense.ID, &expense.AuthorID, &expense.PayerID,
			&descriptionSealed, &amountSealed, &keyID, &expense.SpentAt, &expense.Version,
		); err != nil {
			return nil, err
		}
		expense.GroupID = groupID

		description, err := s.content.OpenString(secure.Sealed{KeyID: keyID, Ciphertext: descriptionSealed}, expenseDescriptionAAD(expense.ID, expense.AuthorID))
		if err != nil {
			return nil, err
		}
		amountRaw, err := s.content.OpenString(secure.Sealed{KeyID: keyID, Ciphertext: amountSealed}, expenseAmountAAD(expense.ID, expense.AuthorID))
		if err != nil {
			return nil, err
		}
		amount, err := strconv.ParseInt(amountRaw, 10, 64)
		if err != nil {
			return nil, err
		}
		expense.Description = description
		expense.AmountMinor = amount

		shares, err := s.sharesOf(ctx, expense.ID)
		if err != nil {
			return nil, err
		}
		expense.Shares = shares
		out = append(out, expense)
	}
	return out, rows.Err()
}

// Balances computes each member's balance over the whole group.
//
// A single snapshot is read, then everything is decrypted and summed in Go: SQL
// cannot sum ciphertext, and a partial page would give a wrong total.
func (s *Service) Balances(ctx context.Context, viewerID, groupID int64) ([]Balance, error) {
	if err := s.requireMember(ctx, viewerID, groupID); err != nil {
		return nil, err
	}

	expenses, err := s.Expenses(ctx, viewerID, groupID)
	if err != nil {
		return nil, err
	}

	paid := map[int64]int64{}
	shared := map[int64]int64{}
	members := map[int64]struct{}{}

	for _, expense := range expenses {
		paid[expense.PayerID] += expense.AmountMinor
		for participant, share := range expense.Shares {
			shared[participant] += share
		}
	}

	rows, err := s.pool.Query(ctx,
		`SELECT user_id FROM split_members WHERE group_id = $1 AND state = 'active'`, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		members[id] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	out := make([]Balance, 0, len(members))
	for id := range members {
		out = append(out, Balance{
			UserID:       id,
			PaidMinor:    paid[id],
			ShareMinor:   shared[id],
			BalanceMinor: BalanceOf(paid[id], shared[id]),
		})
	}
	return out, nil
}

func (s *Service) sharesOf(ctx context.Context, expenseID int64) (map[int64]int64, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT user_id, amount_sealed, key_id FROM split_shares WHERE expense_id = $1`, expenseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := map[int64]int64{}
	for rows.Next() {
		var userID int64
		var sealed []byte
		var keyID string
		if err := rows.Scan(&userID, &sealed, &keyID); err != nil {
			return nil, err
		}
		raw, err := s.content.OpenString(secure.Sealed{KeyID: keyID, Ciphertext: sealed}, shareAAD(expenseID, userID))
		if err != nil {
			return nil, err
		}
		value, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return nil, err
		}
		out[userID] = value
	}
	return out, rows.Err()
}

func (s *Service) requireMember(ctx context.Context, viewerID, groupID int64) error {
	var member bool
	err := s.pool.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM split_members
		                 WHERE group_id = $1 AND user_id = $2 AND state = 'active')`,
		groupID, viewerID,
	).Scan(&member)
	if err != nil {
		return err
	}
	if !member {
		// Neutral answer: a stranger learns nothing about the group.
		return httpapi.NotFound("Группа расходов не найдена")
	}
	return nil
}

func groupTitleAAD(groupID int64) string {
	return "split_group_title:" + strconv.FormatInt(groupID, 10)
}

func expenseDescriptionAAD(expenseID, authorID int64) string {
	return "split_expense_description:" + strconv.FormatInt(expenseID, 10) + ":author:" + strconv.FormatInt(authorID, 10)
}

func expenseAmountAAD(expenseID, authorID int64) string {
	return "split_expense_amount:" + strconv.FormatInt(expenseID, 10) + ":author:" + strconv.FormatInt(authorID, 10)
}

func shareAAD(expenseID, userID int64) string {
	return "split_share:" + strconv.FormatInt(expenseID, 10) + ":user:" + strconv.FormatInt(userID, 10)
}
