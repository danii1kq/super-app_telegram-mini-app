package split

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/jackc/pgx/v5"

	"github.com/danii1kq/srunal/internal/platform/secure"
)

// DebtNotice is the minimal payload a delivery needs.
//
// It carries the amount and the side of the debt only: no receipt photo, no
// other people's expenses, no group content. The worker seals and sends it; it
// never reads the expense tables.
type DebtNotice struct {
	ReminderVersion int `json:"-"`
	// Text is the message body, already localized to Russian for v1.
	Text string `json:"text"`
	// ButtonURL opens the debt in the Mini App.
	ButtonURL string `json:"button_url"`
	// ButtonText is the label of that button.
	ButtonText string `json:"button_text"`
}

// enqueueNotice writes one outbox row inside the caller's transaction.
//
// The row is committed with the debt or repayment, so a rolled-back financial
// operation never leaves a message behind. dedupe_key makes a repeated
// operation a no-op instead of a second message.
func (s *Service) enqueueNotice(ctx context.Context, tx pgx.Tx, userID int64, kind, dedupeKey, subject string, version int64, notice DebtNotice) error {
	// Respect the person's settings: the category can be turned off entirely.
	var enabled, showAmounts bool
	err := tx.QueryRow(ctx,
		`SELECT split_debt_notifications, split_debt_show_amounts
		   FROM user_settings WHERE user_id = $1`, userID,
	).Scan(&enabled, &showAmounts)
	if err != nil && err != pgx.ErrNoRows {
		return err
	}
	if !enabled {
		return nil
	}
	if !showAmounts {
		// Without amounts the message still says a debt changed, but not how much.
		notice = DebtNotice{
			ReminderVersion: notice.ReminderVersion,
			Text:            "SruSplit: изменилась сумма долга. Откройте приложение, чтобы посмотреть детали.",
			ButtonURL:       notice.ButtonURL,
			ButtonText:      "Открыть",
		}
	}

	payload, err := json.Marshal(notice)
	if err != nil {
		return err
	}
	sealed, err := s.delivery.Seal(payload, noticeAAD(userID, dedupeKey))
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO notification_delivery (user_id, kind, payload_sealed, payload_key_id, dedupe_key, subject, subject_version,reminder_version)
		 VALUES ($1, $2, $3, $4, $5, $6, $7,nullif($8,0))
		 ON CONFLICT (user_id, dedupe_key) DO NOTHING`,
		userID, kind, sealed.Ciphertext, sealed.KeyID, dedupeKey, subject, version, notice.ReminderVersion,
	)
	return err
}

// noticeAAD binds a payload to its recipient and event.
func noticeAAD(userID int64, dedupeKey string) string {
	return "split_notice:" + strconv.FormatInt(userID, 10) + ":" + dedupeKey
}

// noticeForDebt builds the message for a debt event.
//
// An expense-backed debt is a confirmed obligation: the expense exists and the
// share is computed. A manual debt has an author and is phrased as their record,
// because the other side has not confirmed anything.
func noticeForDebt(debt Debt, recipientID int64, debtorName, creditorName string, outstanding int64) DebtNotice {
	button := DebtNotice{
		ButtonURL:  "/apps/split/" + strconv.FormatInt(debt.GroupID, 10) + "/debts",
		ButtonText: "Открыть долг",
	}

	// The recipient is the debtor: they owe money.
	if recipientID == debt.DebtorID {
		if debt.SourceExpenseID != nil {
			button.Text = "SruSplit: вы должны " + creditorName + " " + formatMoney(outstanding) + ".\nОснование: общий чек."
		} else {
			button.Text = "SruSplit: по записи " + creditorName + " ваш долг перед ним составляет " + formatMoney(outstanding) + "."
			button.ButtonText = "Открыть запись"
		}
		return button
	}

	// The recipient is the creditor: they are owed money.
	if debt.SourceExpenseID != nil {
		button.Text = "SruSplit: " + debtorName + " должен вам " + formatMoney(outstanding) + ".\nОснование: общий чек."
	} else {
		button.Text = "SruSplit: вы записали долг " + debtorName + " на " + formatMoney(outstanding) + "."
		button.ButtonText = "Открыть запись"
	}
	return button
}

// formatMoney renders minor units as tenge with a space and two decimals.
func formatMoney(minor int64) string {
	sign := ""
	if minor < 0 {
		sign = "-"
		minor = -minor
	}
	whole := minor / 100
	cents := minor % 100
	return sign + groupThousands(whole) + "," + pad2(cents) + " ₸"
}

func groupThousands(value int64) string {
	digits := strconv.FormatInt(value, 10)
	out := make([]byte, 0, len(digits)+len(digits)/3)
	for index, char := range digits {
		if index > 0 && (len(digits)-index)%3 == 0 {
			out = append(out, ' ')
		}
		out = append(out, byte(char))
	}
	return string(out)
}

func pad2(value int64) string {
	if value < 10 {
		return "0" + strconv.FormatInt(value, 10)
	}
	return strconv.FormatInt(value, 10)
}

// sealForDelivery is used by tests to prove the payload is not plaintext.
func sealForDelivery(cipher *secure.Cipher, payload []byte, aad string) (secure.Sealed, error) {
	return cipher.Seal(payload, aad)
}
