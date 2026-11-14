package notify

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/danii1kq/srunal/internal/core/auth"
	"github.com/danii1kq/srunal/internal/platform/postgres"
	"github.com/danii1kq/srunal/internal/platform/secure"
	"github.com/danii1kq/srunal/internal/platform/telegram"
	"github.com/jackc/pgx/v5/pgxpool"
)

type leasedBot struct {
	mu       sync.Mutex
	messages []telegram.SendMessageRequest
	failure  error
}

func (b *leasedBot) SendMessage(_ context.Context, r telegram.SendMessageRequest) (int64, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.messages = append(b.messages, r)
	return 1, b.failure
}

func leaseFixture(t *testing.T) (*Worker, *leasedBot, func(string) int64) {
	t.Helper()
	dsn := os.Getenv("SRUNAL_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set SRUNAL_TEST_DATABASE_URL")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	schema := fmt.Sprintf("notify_lease_%d", time.Now().UnixNano())
	if _, err = admin.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { admin.Exec(context.Background(), "DROP SCHEMA "+schema+" CASCADE"); admin.Close() })
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	for _, name := range []string{"000001_core_identity.up.sql", "000004_session_initdata.up.sql", "000006_split.up.sql", "000009_split_receipts_debts.up.sql", "000010_split_report_media_notify.up.sql", "000018_split_payment_confirmation.up.sql", "000020_split_reminders.up.sql", "000030_tinder_notifications.up.sql"} {
		raw, err := os.ReadFile("../../../../db/migrations/" + name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = pool.Exec(ctx, string(raw)); err != nil {
			t.Fatal(err)
		}
	}
	cipher, err := secure.NewCipher(base64.StdEncoding.EncodeToString(make([]byte, 32)))
	if err != nil {
		t.Fatal(err)
	}
	identity, err := cipher.SealString("12345", "user:telegram_id")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO users(telegram_id_lookup,telegram_id_sealed,key_id,public_name)VALUES('test',$1,$2,'Tester')`, identity.Ciphertext, identity.KeyID); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO user_settings(user_id,bot_notifications_opt_in)VALUES(1,true)`); err != nil {
		t.Fatal(err)
	}
	bot := &leasedBot{}
	w := NewWorker(&postgres.Pool{Pool: pool}, bot, cipher, cipher, telegram.MiniAppURL("srunal_test_bot", ""))
	w.batch = 1
	enqueue := func(key string) int64 {
		raw, err := json.Marshal(payload{Text: "SruSplit: 2 500,00 ₸", ButtonURL: "/apps/split/1/debts", ButtonText: "Open"})
		if err != nil {
			t.Fatal(err)
		}
		sealed, err := cipher.Seal(raw, noticeAAD(1, key))
		if err != nil {
			t.Fatal(err)
		}
		var id int64
		if err = pool.QueryRow(ctx, `INSERT INTO notification_delivery(user_id,kind,payload_sealed,payload_key_id,dedupe_key)VALUES(1,'split.test',$1,$2,$3)RETURNING id`, sealed.Ciphertext, sealed.KeyID, key).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	return w, bot, enqueue
}

func TestWorkersLeaseAndFenceExpiredClaims(t *testing.T) {
	w, bot, enqueue := leaseFixture(t)
	ctx := context.Background()
	id := enqueue("one")
	other := NewWorker(w.pool, bot, w.delivery, w.identity, w.miniAppURL)
	other.batch = 1
	var wg sync.WaitGroup
	var a, b []job
	var errA, errB error
	wg.Add(2)
	go func() { defer wg.Done(); a, errA = w.claim(ctx) }()
	go func() { defer wg.Done(); b, errB = other.claim(ctx) }()
	wg.Wait()
	if errA != nil || errB != nil || len(a)+len(b) != 1 {
		t.Fatal(a, b, errA, errB)
	}
	old := append(a, b...)[0]
	if _, err := w.pool.Exec(ctx, `UPDATE notification_delivery SET lease_until=now()-interval '1 second' WHERE id=$1`, id); err != nil {
		t.Fatal(err)
	}
	current, err := other.claim(ctx)
	if err != nil || len(current) != 1 {
		t.Fatal(current, err)
	}
	w.deliver(ctx, old)
	if len(bot.messages) != 0 {
		t.Fatal("expired owner sent a message")
	}
	if _, err = w.pool.Exec(ctx, `UPDATE user_settings SET split_debt_show_amounts=false WHERE user_id=1`); err != nil {
		t.Fatal(err)
	}
	other.deliver(ctx, current[0])
	if len(bot.messages) != 1 || strings.Contains(bot.messages[0].Text, "500") || bot.messages[0].ReplyMarkup.InlineKeyboard[0][0].URL != "https://t.me/srunal_test_bot?startapp=split_debts_1" {
		t.Fatal(bot.messages)
	}
	var state string
	if err = w.pool.QueryRow(ctx, `SELECT state FROM notification_delivery WHERE id=$1`, id).Scan(&state); err != nil || state != "sent" {
		t.Fatal(state, err)
	}
}

func TestWorkerRetriesAndChecksLatestPreference(t *testing.T) {
	w, bot, enqueue := leaseFixture(t)
	ctx := context.Background()
	id := enqueue("retry")
	jobs, err := w.claim(ctx)
	if err != nil || len(jobs) != 1 {
		t.Fatal(jobs, err)
	}
	bot.failure = &telegram.APIError{Description: "Too Many Requests", RetryAfter: 30}
	w.deliver(ctx, jobs[0])
	var attempts int
	var lease *time.Time
	if err = w.pool.QueryRow(ctx, `SELECT attempts,lease_until FROM notification_delivery WHERE id=$1`, id).Scan(&attempts, &lease); err != nil || attempts != 1 || lease != nil {
		t.Fatal(attempts, lease, err)
	}
	if _, err = w.pool.Exec(ctx, `UPDATE notification_delivery SET next_attempt_at=now() WHERE id=$1`, id); err != nil {
		t.Fatal(err)
	}
	jobs, err = w.claim(ctx)
	if err != nil || len(jobs) != 1 {
		t.Fatal(jobs, err)
	}
	if _, err = w.pool.Exec(ctx, `UPDATE user_settings SET bot_notifications_opt_in=false WHERE user_id=1`); err != nil {
		t.Fatal(err)
	}
	w.deliver(ctx, jobs[0])
	if len(bot.messages) != 1 {
		t.Fatal("ignored latest notification preference")
	}
	var state string
	if err = w.pool.QueryRow(ctx, `SELECT state FROM notification_delivery WHERE id=$1`, id).Scan(&state); err != nil || state != "skipped" {
		t.Fatal(state, err)
	}
}

func TestIdentityMigrationAndLoginKeepContentKeyAwayFromWorker(t *testing.T) {
	w, bot, enqueue := leaseFixture(t)
	ctx := context.Background()
	rawKey := make([]byte, 32)
	rawKey[0] = 19
	identity, err := secure.NewCipher(base64.StdEncoding.EncodeToString(rawKey))
	if err != nil {
		t.Fatal(err)
	}
	index, err := secure.NewLookupIndex(base64.StdEncoding.EncodeToString(rawKey))
	if err != nil {
		t.Fatal(err)
	}
	service := auth.NewService(w.pool, index, w.delivery, time.Hour, identity)
	if err = service.MigrateIdentityKeys(ctx); err != nil {
		t.Fatal(err)
	}
	if err = service.MigrateIdentityKeys(ctx); err != nil {
		t.Fatal("migration replay", err)
	}
	var raw []byte
	var key string
	if err = w.pool.QueryRow(ctx, `SELECT telegram_id_sealed,key_id FROM users WHERE id=1`).Scan(&raw, &key); err != nil {
		t.Fatal(err)
	}
	plain, err := identity.OpenString(secure.Sealed{Ciphertext: raw, KeyID: key}, "user:telegram_id")
	if err != nil || plain != "12345" {
		t.Fatal("existing identity not migrated", err)
	}
	if _, err = w.delivery.OpenString(secure.Sealed{Ciphertext: raw, KeyID: key}, "user:telegram_id"); err == nil {
		t.Fatal("content key can still read migrated identity")
	}
	user, token, err := service.Login(ctx, auth.InitData{TelegramID: 6789, FirstName: "Identity test", Raw: "signed-test-data"})
	if err != nil {
		t.Fatal(err)
	}
	if err = w.pool.QueryRow(ctx, `SELECT telegram_id_sealed,key_id FROM users WHERE id=$1`, user.ID).Scan(&raw, &key); err != nil {
		t.Fatal(err)
	}
	plain, err = identity.OpenString(secure.Sealed{Ciphertext: raw, KeyID: key}, "user:telegram_id")
	if err != nil || plain != "6789" {
		t.Fatal("new identity not isolated", err)
	}
	if value, err := service.InitDataFor(ctx, user.ID, token); err != nil || value != "signed-test-data" {
		t.Fatal("content session broke", err)
	}
	w.identity = identity
	enqueue("after-migration")
	jobs, err := w.claim(ctx)
	if err != nil || len(jobs) != 1 {
		t.Fatal(jobs, err)
	}
	w.deliver(ctx, jobs[0])
	if len(bot.messages) != 1 || bot.messages[0].ChatID != 12345 {
		t.Fatal("worker could not deliver migrated identity", bot.messages)
	}
}

func TestTinderDeliveryUsesItsOwnCipherContextAndPreferences(t *testing.T) {
	w, bot, _ := leaseFixture(t)
	ctx := context.Background()
	if _, err := w.pool.Exec(ctx, `UPDATE user_settings SET split_debt_notifications=false,split_debt_show_amounts=false WHERE user_id=1`); err != nil {
		t.Fatal(err)
	}
	enqueue := func(key string) {
		plain, err := json.Marshal(payload{Text: "SruTinder: match with Tester", ButtonURL: "/apps/tinder", ButtonText: "Open Tinder"})
		if err != nil {
			t.Fatal(err)
		}
		sealed, err := w.delivery.Seal(plain, "tinder_notice:1:"+key)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = w.pool.Exec(ctx, `INSERT INTO notification_delivery(user_id,kind,payload_sealed,payload_key_id,dedupe_key)VALUES(1,'tinder.matched',$1,$2,$3)`, sealed.Ciphertext, sealed.KeyID, key); err != nil {
			t.Fatal(err)
		}
	}
	enqueue("match:first")
	if err := w.tick(ctx); err != nil {
		t.Fatal(err)
	}
	if len(bot.messages) != 1 || bot.messages[0].Text != "SruTinder: match with Tester" || bot.messages[0].ReplyMarkup.InlineKeyboard[0][0].URL != "https://t.me/srunal_test_bot?startapp=tinder" {
		t.Fatal(bot.messages)
	}
	enqueue("match:second")
	if _, err := w.pool.Exec(ctx, `UPDATE user_settings SET tinder_match_notifications=false WHERE user_id=1`); err != nil {
		t.Fatal(err)
	}
	if err := w.tick(ctx); err != nil {
		t.Fatal(err)
	}
	if len(bot.messages) != 1 {
		t.Fatal("ignored Tinder notification preference")
	}
}
