// Package notify owns the notification outbox and the Telegram delivery worker.
package notify

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/danii1kq/srunal/internal/platform/postgres"
	"github.com/danii1kq/srunal/internal/platform/secure"
	"github.com/danii1kq/srunal/internal/platform/telegram"
)

// Sender is the Bot API surface the worker needs. A test double implements it.
type Sender interface {
	SendMessage(ctx context.Context, req telegram.SendMessageRequest) (int64, error)
}

// Worker drains notification_delivery rows and sends them through the Bot API.
type Worker struct {
	pool     *postgres.Pool
	bot      Sender
	delivery *secure.Cipher
	// identity opens the sealed Telegram id; in production it is a different
	// key from the delivery key.
	identity *secure.Cipher
	// batch bounds one tick so a backlog never holds the pool.
	batch int
	// now is injectable so tests can control retry timing.
	now func() time.Time
}

func NewWorker(pool *postgres.Pool, bot Sender, delivery, identity *secure.Cipher) *Worker {
	return &Worker{pool: pool, bot: bot, delivery: delivery, identity: identity, batch: 20, now: time.Now}
}

// Run polls the outbox until ctx is cancelled.
func (w *Worker) Run(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.tick(ctx); err != nil {
				slog.Error("notify tick", "err", err)
			}
		}
	}
}

// payload is the sealed body a delivery carries.
type payload struct {
	Text       string `json:"text"`
	ButtonURL  string `json:"button_url"`
	ButtonText string `json:"button_text"`
}

// job is one claimed row.
type job struct {
	id         int64
	userID     int64
	kind       string
	sealed     []byte
	keyID      string
	dedupeKey  string
	subject    *string
	version    *int64
	attempts   int
}

// tick claims a batch and sends it.
//
// Claiming uses FOR UPDATE SKIP LOCKED so two workers never send the same row.
// A row is marked sent, skipped or failed; a temporary failure is retried with
// a delay Telegram asked for.
func (w *Worker) tick(ctx context.Context) error {
	jobs, err := w.claim(ctx)
	if err != nil {
		return err
	}
	for _, item := range jobs {
		w.deliver(ctx, item)
	}
	return nil
}

func (w *Worker) claim(ctx context.Context) ([]job, error) {
	rows, err := w.pool.Query(ctx,
		`SELECT id, user_id, kind, payload_sealed, payload_key_id, dedupe_key, subject, subject_version, attempts
		   FROM notification_delivery
		  WHERE state = 'pending' AND next_attempt_at <= now()
		  ORDER BY next_attempt_at, id
		  LIMIT $1
		  FOR UPDATE SKIP LOCKED`, w.batch)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]job, 0, w.batch)
	for rows.Next() {
		var item job
		if err := rows.Scan(&item.id, &item.userID, &item.kind, &item.sealed, &item.keyID,
			&item.dedupeKey, &item.subject, &item.version, &item.attempts); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

// deliver sends one row and records the outcome.
func (w *Worker) deliver(ctx context.Context, item job) {
	// A newer change to the same subject makes an older unsent message stale:
	// sending "you owe 4000" after the debt was settled would be wrong.
	if item.subject != nil {
		var newer bool
		if err := w.pool.QueryRow(ctx,
			`SELECT EXISTS (
			   SELECT 1 FROM notification_delivery
			    WHERE subject = $1 AND user_id = $2 AND state = 'pending'
			      AND coalesce(subject_version, 0) > coalesce($3, 0))`,
			*item.subject, item.userID, item.version,
		).Scan(&newer); err != nil {
			w.fail(ctx, item, "lookup: "+err.Error(), time.Minute)
			return
		}
		if newer {
			w.mark(ctx, item.id, "skipped", "superseded by a newer change")
			return
		}
	}

	// The chat id is the Telegram id of the person, sealed in the identity table.
	chatID, err := w.chatID(ctx, item.userID)
	if err != nil {
		// No linked Telegram chat: nothing to send to, and retrying will not help.
		w.mark(ctx, item.id, "skipped", "no telegram chat")
		return
	}

	raw, err := w.delivery.Open(secure.Sealed{KeyID: item.keyID, Ciphertext: item.sealed},
		noticeAAD(item.userID, item.dedupeKey))
	if err != nil {
		w.mark(ctx, item.id, "failed", "payload cannot be opened")
		return
	}
	var body payload
	if err := json.Unmarshal(raw, &body); err != nil {
		w.mark(ctx, item.id, "failed", "payload is malformed")
		return
	}

	request := telegram.SendMessageRequest{
		ChatID: chatID,
		Text:   body.Text,
	}
	if body.ButtonURL != "" {
		request.ReplyMarkup = &telegram.InlineMarkup{
			InlineKeyboard: [][]telegram.InlineButton{{
				{Text: body.ButtonText, URL: body.ButtonURL},
			}},
		}
	}

	_, err = w.bot.SendMessage(ctx, request)
	if err == nil {
		w.mark(ctx, item.id, "sent", "")
		return
	}

	var apiErr *telegram.APIError
	if errors.As(err, &apiErr) {
		if apiErr.RetryAfter > 0 {
			// Telegram asked to slow down: honour it instead of hammering.
			w.fail(ctx, item, "rate limited", time.Duration(apiErr.RetryAfter)*time.Second)
			return
		}
		if isPermanent(apiErr.Description) {
			// The bot is blocked or the chat does not exist: retrying is pointless.
			w.mark(ctx, item.id, "skipped", "telegram refused: "+apiErr.Description)
			return
		}
	}

	// A network error or a 5xx: retry with backoff. Exactly-once is not promised
	// when the answer is lost, so a duplicate is possible and acceptable.
	backoff := time.Minute << min(item.attempts, 6)
	w.fail(ctx, item, "send failed", backoff)
}

// isPermanent reports whether retrying cannot help.
func isPermanent(description string) bool {
	switch description {
	case "Forbidden: bot was blocked by the user",
		"Forbidden: user is deactivated",
		"Bad Request: chat not found",
		"Forbidden: bot can't initiate conversation with a user":
		return true
	}
	return false
}

func (w *Worker) mark(ctx context.Context, id int64, state, reason string) {
	_, err := w.pool.Exec(ctx,
		`UPDATE notification_delivery
		    SET state = $2, last_error = nullif($3, ''), sent_at = case when $2 = 'sent' then now() else sent_at end
		  WHERE id = $1`, id, state, reason)
	if err != nil {
		slog.Error("notify: mark", "err", err)
	}
}

func (w *Worker) fail(ctx context.Context, item job, reason string, delay time.Duration) {
	_, err := w.pool.Exec(ctx,
		`UPDATE notification_delivery
		    SET attempts = attempts + 1,
		        last_error = $2,
		        next_attempt_at = $3,
		        state = case when attempts + 1 >= 8 then 'failed' else 'pending' end
		  WHERE id = $1`, item.id, reason, w.now().Add(delay))
	if err != nil {
		slog.Error("notify: fail", "err", err)
	}
}

// chatID returns the Telegram id of a person, or an error when there is none.
//
// The id is sealed in the identity table; the worker opens it with the identity
// key it is given, not with the content key.
func (w *Worker) chatID(ctx context.Context, userID int64) (int64, error) {
	var sealed []byte
	var keyID string
	err := w.pool.QueryRow(ctx,
		`SELECT telegram_id_sealed, key_id FROM users WHERE id = $1 AND status = 'active'`, userID,
	).Scan(&sealed, &keyID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, errors.New("notify: no such user")
		}
		return 0, err
	}
	raw, err := w.identity.Open(secure.Sealed{KeyID: keyID, Ciphertext: sealed}, "user:telegram_id")
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(string(raw), 10, 64)
}

func noticeAAD(userID int64, dedupeKey string) string {
	return "split_notice:" + strconv.FormatInt(userID, 10) + ":" + dedupeKey
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
