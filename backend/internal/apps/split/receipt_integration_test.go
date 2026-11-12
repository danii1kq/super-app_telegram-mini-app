package split

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/danii1kq/srunal/internal/core/auth"
	"github.com/danii1kq/srunal/internal/platform/httpapi"
	"github.com/danii1kq/srunal/internal/platform/ocr"
	"github.com/danii1kq/srunal/internal/platform/postgres"
	"github.com/danii1kq/srunal/internal/platform/secure"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type receiptTestReader struct{ calls atomic.Int32 }

func (r *receiptTestReader) Read(_ context.Context, raw []byte, mime string) (ocr.Result, error) {
	r.calls.Add(1)
	if mime != "image/jpeg" || len(raw) < 2 || raw[0] != 255 || raw[1] != 216 {
		return ocr.Result{}, ocr.ErrUnreadable
	}
	return ocr.Result{Currency: "KZT", Total: 600000, Items: []ocr.Item{{Name: "Пицца", Quantity: 1, Amount: 400000}, {Name: "Бургер", Quantity: 1, Amount: 200000}}}, nil
}

func receiptTestService(t *testing.T) (*Service, *receiptTestReader) {
	t.Helper()
	dsn := os.Getenv("SRUNAL_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set SRUNAL_TEST_DATABASE_URL for isolated PostgreSQL receipt tests")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	schema := pgx.Identifier{"split_receipt_" + strconv.FormatInt(time.Now().UnixNano(), 10)}.Sanitize()
	if _, err = admin.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		admin.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() { admin.Exec(context.Background(), "DROP SCHEMA "+schema+" CASCADE"); admin.Close() })
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	cfg.MaxConns = 10
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	_, err = pool.Exec(ctx, `CREATE TABLE users(id bigint PRIMARY KEY, public_name text NOT NULL, username text, status text NOT NULL DEFAULT 'active');
 CREATE TABLE user_settings(user_id bigint PRIMARY KEY);
 CREATE TABLE sessions(token_hash text PRIMARY KEY,user_id bigint,revoked_at timestamptz,expires_at timestamptz);
 INSERT INTO users(id,public_name) VALUES(1,'Alice'),(2,'Bob'),(3,'Eve');`)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"000006_split.up.sql", "000009_split_receipts_debts.up.sql", "000010_split_report_media_notify.up.sql", "000014_split_deepseek_budget.up.sql"} {
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
	reader := &receiptTestReader{}
	return NewService(&postgres.Pool{Pool: pool}, cipher, reader, cipher), reader
}

func testReceiptImage(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 120, 80))
	img.Set(0, 0, color.White)
	var out bytes.Buffer
	if err := png.Encode(&out, img); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}
func receiptTestMux(t *testing.T, s *Service) *http.ServeMux {
	t.Helper()
	for id := 1; id <= 3; id++ {
		_, err := s.pool.Exec(context.Background(), `INSERT INTO sessions(token_hash,user_id,expires_at)VALUES($1,$2,now()+interval '1 hour')`, secure.TokenHash(fmt.Sprintf("test-%d", id)), id)
		if err != nil {
			t.Fatal(err)
		}
	}
	a := auth.NewService(s.pool, nil, s.content, time.Hour)
	mux := http.NewServeMux()
	NewHandler(s, a, s.pool).Register(mux)
	return mux
}
func requireReceiptStatus(t *testing.T, err error, status int) {
	t.Helper()
	var e *httpapi.Error
	if !errors.As(err, &e) || e.Status != status {
		t.Fatalf("status%d got%v", status, err)
	}
}

func TestReceiptImportPrivacyBudgetAndConsent(t *testing.T) {
	s, reader := receiptTestService(t)
	ctx := context.Background()
	group, err := s.CreateGroup(ctx, 1, "Dinner", []int64{2})
	if err != nil {
		t.Fatal(err)
	}
	mux := receiptTestMux(t, s)
	raw := testReceiptImage(t)
	send := func(token string, consent bool, body []byte) *httptest.ResponseRecorder {
		r := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/split/groups/%d/receipts", group.ID), bytes.NewReader(body))
		r.Header.Set("Authorization", "Bearer "+token)
		r.Header.Set("Content-Type", "image/png")
		if consent {
			r.Header.Set("X-AI-Consent", "deepseek")
		}
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		return w
	}
	if w := send("test-1", false, raw); w.Code != 400 {
		t.Fatalf("consent: %d %s", w.Code, w.Body)
	}
	if w := send("test-3", true, raw); w.Code != 404 {
		t.Fatalf("outsider: %d", w.Code)
	}
	if reader.calls.Load() != 0 {
		t.Fatal("provider called before access and consent")
	}
	w := send("test-1", true, raw)
	if w.Code != 201 {
		t.Fatalf("import: %d %s", w.Code, w.Body)
	}
	var body struct {
		ID    string `json:"id"`
		Items []any  `json:"items"`
	}
	if json.Unmarshal(w.Body.Bytes(), &body) != nil || len(body.Items) != 2 {
		t.Fatal("missing draft")
	}
	var stored []byte
	if err = s.pool.QueryRow(ctx, `SELECT image_sealed FROM split_receipts WHERE id=$1`, body.ID).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(stored, raw) || bytes.HasPrefix(stored, []byte{255, 216}) {
		t.Fatal("receipt photo stored as plaintext")
	}
	for token, want := range map[string]int{"test-2": 200, "test-3": 404} {
		r := httptest.NewRequest("GET", "/api/v1/split/receipts/"+body.ID+"/image", nil)
		r.Header.Set("Authorization", "Bearer "+token)
		imageResponse := httptest.NewRecorder()
		mux.ServeHTTP(imageResponse, r)
		if imageResponse.Code != want {
			t.Fatalf("image rights: %d", imageResponse.Code)
		}
		if want == 200 && (imageResponse.Header().Get("Cache-Control") != "private, no-store" || imageResponse.Header().Get("X-Content-Type-Options") != "nosniff") {
			t.Fatal("image caching")
		}
	}
	if w = send("test-1", true, raw); w.Code != 429 {
		t.Fatalf("cooldown: %d %s", w.Code, w.Body)
	}
	if reader.calls.Load() != 1 {
		t.Fatal("cooldown called provider")
	}
	if _, err = s.pool.Exec(ctx, `WITH aged AS (UPDATE split_receipts SET created_at=now()-interval '1 minute' RETURNING id) INSERT INTO split_receipts(group_id,author_id,image_sealed,image_key_id,image_mime,image_bytes,created_at) SELECT $1,1,''::bytea,'fixture','image/jpeg',1,now()-interval '1 minute' FROM generate_series(1,49)`, group.ID); err != nil {
		t.Fatal(err)
	}
	if w = send("test-1", true, raw); w.Code != 429 {
		t.Fatalf("dailybudget: %d", w.Code)
	}
}

func TestReceiptConcurrentRecognitionBudget(t *testing.T) {
	s, reader := receiptTestService(t)
	ctx := context.Background()
	group, err := s.CreateGroup(ctx, 1, "Dinner", nil)
	if err != nil {
		t.Fatal(err)
	}
	raw := testReceiptImage(t)
	var wg sync.WaitGroup
	var successes atomic.Int32
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := s.CreateReceipt(ctx, 1, group.ID, raw, "image/png")
			if err == nil {
				successes.Add(1)
			} else {
				var e *httpapi.Error
				if !errors.As(err, &e) || e.Status != 429 {
					t.Errorf("unexpectedimport: %v", err)
				}
			}
		}()
	}
	wg.Wait()
	if successes.Load() != 1 || reader.calls.Load() != 1 {
		t.Fatalf("paid attempts %d provider%d", successes.Load(), reader.calls.Load())
	}
}

func TestReceiptDraftLimits(t *testing.T) {
	s, _ := receiptTestService(t)
	ctx := context.Background()
	g, err := s.CreateGroup(ctx, 1, "Dinner", []int64{2})
	if err != nil {
		t.Fatal(err)
	}
	r, err := s.CreateManualReceipt(ctx, 1, g.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range []ReceiptItem{{Name: "x", Quantity: 1000, AmountMinor: 100}, {Name: strings.Repeat("x", 201), Quantity: 1, AmountMinor: 100}, {Name: "x", Quantity: 1, AmountMinor: ocr.MaxMoneyMinor + 1}, {Name: "x", Quantity: 1, AmountMinor: 100, Units: map[int64]int{1: 1000}}} {
		_, err = s.UpdateReceipt(ctx, 1, r.ID, 100, 0, 0, []ReceiptItem{item})
		requireReceiptStatus(t, err, 422)
	}
}
