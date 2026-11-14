package day

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"hash/crc32"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/danii1kq/srunal/internal/core/auth"
	"github.com/danii1kq/srunal/internal/platform/postgres"
	"github.com/danii1kq/srunal/internal/platform/secure"
	"github.com/danii1kq/srunal/internal/platform/vision"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestTimetablePreviewValidatesWeekTimesAndUncertainty(t *testing.T) {
	week, zone, err := importWeek("2026-03-02", "America/New_York")
	if err != nil {
		t.Fatal(err)
	}
	items, err := timetableItems(json.RawMessage(`{"items":[
	 {"title":" Algebra ","date":"2026-03-02","start":"10:00","end":"10:01","location":" A1 ","uncertain":false},
	 {"title":"DST gap","date":"2026-03-08","start":"02:30","end":"03:30"},
	 {"title":"Wrong week","date":"2026-03-09","start":"10:00","end":"11:00"},
	 {"title":"Unknown end","date":"2026-03-03","start":"10:00","end":""},
	 {"title":"No inferred overnight","date":"2026-03-03","start":"23:30","end":"00:30"}
	]}`), week, zone)
	if err != nil || len(items) != 5 {
		t.Fatalf("items: %#v %v", items, err)
	}
	if items[0].Title != "Algebra" || items[0].Location != "A1" || items[0].Uncertain || items[0].StartsAt != "2026-03-02T10:00:00-05:00" || items[0].EndsAt != "2026-03-02T10:01:00-05:00" {
		t.Fatalf("valid minute: %#v", items[0])
	}
	for _, index := range []int{1, 2} {
		if !items[index].Uncertain || items[index].StartsAt != "" || items[index].EndsAt != "" {
			t.Fatalf("invented date: %#v", items[index])
		}
	}
	for _, index := range []int{3, 4} {
		if !items[index].Uncertain || items[index].EndsAt != "" {
			t.Fatalf("invented end time: %#v", items[index])
		}
	}
	if _, _, err := importWeek("2026-03-03", "UTC"); err == nil {
		t.Fatal("non-Monday accepted")
	}
	if _, _, err := importWeek("2026-03-02", "bad/zone"); err == nil {
		t.Fatal("invalid timezone accepted")
	}
	for _, zone := range []string{"", "Local"} {
		if _, _, err := importWeek("2026-03-02", zone); err == nil {
			t.Fatal("implicit timezone accepted")
		}
	}
	if !strings.Contains(timetablePrompt(week, zone.String()), "Do not create repeated weeks") {
		t.Fatal("missing explicit recurrence boundary")
	}
	tooMany := make([]map[string]string, 41)
	raw, _ := json.Marshal(map[string]any{"items": tooMany})
	if _, err := timetableItems(raw, week, zone); err == nil {
		t.Fatal("unbounded preview accepted")
	}
	oversized, _ := json.Marshal(map[string]any{"items": []map[string]any{{"title": strings.Repeat("a", 201), "date": "2026-03-02", "start": "10:00", "end": "11:00"}}})
	long, err := timetableItems(oversized, week, zone)
	if err != nil || len(long[0].Title) != 200 || !long[0].Uncertain {
		t.Fatal("truncated content must be marked uncertain")
	}
}

func TestTimetableImageBoundsBeforeQuotaOrProvider(t *testing.T) {
	var imageBytes bytes.Buffer
	if err := png.Encode(&imageBytes, image.NewNRGBA(image.Rect(0, 0, 40, 20))); err != nil {
		t.Fatal(err)
	}
	if result, err := prepareTimetableImage(imageBytes.Bytes()); err != nil || result.Mime != "image/jpeg" {
		t.Fatal("valid photo failed", err)
	}
	imageBytes.Reset()
	if err := png.Encode(&imageBytes, image.NewNRGBA(image.Rect(0, 0, 9000, 1))); err != nil {
		t.Fatal(err)
	}
	if _, err := prepareTimetableImage(imageBytes.Bytes()); err == nil || !strings.Contains(err.Error(), "8192") {
		t.Fatal("long narrow image reached provider", err)
	}
	var header bytes.Buffer
	header.Write([]byte{137, 80, 78, 71, 13, 10, 26, 10})
	binary.Write(&header, binary.BigEndian, uint32(13))
	chunk := append([]byte("IHDR"), make([]byte, 13)...)
	binary.BigEndian.PutUint32(chunk[4:8], 8000)
	binary.BigEndian.PutUint32(chunk[8:12], 3000)
	chunk[12], chunk[13] = 8, 6
	header.Write(chunk)
	binary.Write(&header, binary.BigEndian, crc32.ChecksumIEEE(chunk))
	if _, err := prepareTimetableImage(header.Bytes()); err == nil || !strings.Contains(err.Error(), "16") {
		t.Fatal("oversized pixel header accepted", err)
	}
}

func importTestService(t *testing.T) *Service {
	t.Helper()
	dsn := os.Getenv("SRUNAL_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set SRUNAL_TEST_DATABASE_URL for isolated PostgreSQL quota test")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	schema := pgx.Identifier{fmt.Sprintf("day_import_%d", time.Now().UnixNano())}.Sanitize()
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		admin.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = admin.Exec(context.Background(), "DROP SCHEMA "+schema+" CASCADE"); admin.Close() })
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	config.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if _, err := pool.Exec(ctx, `CREATE TABLE users (id bigint PRIMARY KEY, public_name text NOT NULL, username text, status text NOT NULL);
	 CREATE TABLE sessions (token_hash text PRIMARY KEY, user_id bigint NOT NULL REFERENCES users(id), revoked_at timestamptz, expires_at timestamptz NOT NULL);
	 CREATE TABLE app_links (user_id bigint NOT NULL REFERENCES users(id), app_id text NOT NULL, external_id text NOT NULL, PRIMARY KEY(user_id,app_id));
	 INSERT INTO users VALUES (1,'Test user',NULL,'active')`); err != nil {
		t.Fatal(err)
	}
	migration, err := os.ReadFile("../../../../db/migrations/000013_day_import_limits.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, string(migration)); err != nil {
		t.Fatal(err)
	}
	return NewService(&postgres.Pool{Pool: pool}, nil)
}

func TestImportQuotaIsAtomicAndResetsByUTCDay(t *testing.T) {
	service := importTestService(t)
	pool := service.pool
	ctx := context.Background()
	var group sync.WaitGroup
	results := make(chan time.Duration, 2)
	failures := make(chan error, 2)
	for range 2 {
		group.Add(1)
		go func() {
			defer group.Done()
			wait, err := service.reserveImport(ctx, 1)
			results <- wait
			failures <- err
		}()
	}
	group.Wait()
	close(results)
	close(failures)
	for err := range failures {
		if err != nil {
			t.Fatal(err)
		}
	}
	allowed := 0
	blocked := 0
	for wait := range results {
		if wait == 0 {
			allowed++
		} else {
			blocked++
		}
	}
	if allowed != 1 || blocked != 1 {
		t.Fatalf("quota race: allowed=%d blocked=%d", allowed, blocked)
	}
	if _, err := pool.Exec(ctx, `UPDATE day_import_limits SET used=19,last_request_at=now()-interval '1 minute'`); err != nil {
		t.Fatal(err)
	}
	if wait, err := service.reserveImport(ctx, 1); err != nil || wait != 0 {
		t.Fatal("20th request should be allowed", wait, err)
	}
	if wait, err := service.reserveImport(ctx, 1); err != nil || wait <= 0 {
		t.Fatal("daily budget must block the next request", wait, err)
	}
	if _, err := pool.Exec(ctx, `UPDATE day_import_limits SET usage_day=(now() AT TIME ZONE 'UTC')::date-1,last_request_at=now()-interval '1 minute'`); err != nil {
		t.Fatal(err)
	}
	if wait, err := service.reserveImport(ctx, 1); err != nil || wait != 0 {
		t.Fatal("new UTC day must reset budget", wait, err)
	}
	var used int
	if err := pool.QueryRow(ctx, `SELECT used FROM day_import_limits`).Scan(&used); err != nil || used != 1 {
		t.Fatal("new-day quota not reset", used, err)
	}
}

type importBodyTrap struct{ reads int }

func (body *importBodyTrap) Read([]byte) (int, error) {
	body.reads++
	return 0, errors.New("body read reached")
}
func (*importBodyTrap) Close() error { return nil }

func TestBusyImportRejectsBeforeReadingOrChargingQuota(t *testing.T) {
	service := importTestService(t)
	token := "day-import-test-not-production-token"
	if _, err := service.pool.Exec(context.Background(), `INSERT INTO sessions (token_hash,user_id,expires_at) VALUES ($1,1,now()+interval '1 hour');`, secure.TokenHash(token)); err != nil {
		t.Fatal(err)
	}
	if _, err := service.pool.Exec(context.Background(), `INSERT INTO app_links VALUES (1,'day','verified-upstream-account')`); err != nil {
		t.Fatal(err)
	}
	authService := auth.NewService(service.pool, nil, nil, time.Hour)
	handler := NewHandler(service, authService, vision.NewClient("http://127.0.0.1:1", "synthetic-test-key", "deepseek-flash"), NewWeatherClient(""))
	mux := http.NewServeMux()
	handler.Register(mux)
	for range cap(handler.importSlots) {
		handler.importSlots <- struct{}{}
	}
	request := func(body *importBodyTrap) *http.Request {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/day/import/preview?week_start=2026-11-09&timezone=UTC", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("X-AI-Consent", "deepseek")
		req.Header.Set("Content-Type", "image/png")
		req.Body = body
		req.ContentLength = 10 << 20
		return req
	}
	body := &importBodyTrap{}
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request(body))
	var result struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if response.Code != http.StatusServiceUnavailable || response.Header().Get("Retry-After") != "5" || result.Error.Code != "import_busy" || body.reads != 0 {
		t.Fatalf("busy upload reached body or returned wrong response: code=%d header=%q error=%q reads=%d", response.Code, response.Header().Get("Retry-After"), result.Error.Code, body.reads)
	}
	if len(handler.importSlots) != cap(handler.importSlots) {
		t.Fatal("busy request released another import's slot")
	}
	for range cap(handler.importSlots) {
		<-handler.importSlots
	}
	body = &importBodyTrap{}
	response = httptest.NewRecorder()
	mux.ServeHTTP(response, request(body))
	if response.Code != http.StatusBadRequest || body.reads != 1 || len(handler.importSlots) != 0 {
		t.Fatal("failed body must release its acquired slot", response.Code, body.reads, len(handler.importSlots))
	}
	var charged int
	if err := service.pool.QueryRow(context.Background(), `SELECT count(*) FROM day_import_limits`).Scan(&charged); err != nil || charged != 0 {
		t.Fatal("busy or malformed upload charged quota", charged, err)
	}
}

func TestSharedVisionBusyIsReturnedAsRetryableImportBusy(t *testing.T) {
	service := importTestService(t)
	token := "shared-vision-busy-not-production-token"
	if _, err := service.pool.Exec(context.Background(), `INSERT INTO sessions (token_hash,user_id,expires_at) VALUES ($1,1,now()+interval '1 hour')`, secure.TokenHash(token)); err != nil {
		t.Fatal(err)
	}
	if _, err := service.pool.Exec(context.Background(), `INSERT INTO app_links VALUES (1,'day','verified-upstream-account')`); err != nil {
		t.Fatal(err)
	}
	release := make(chan struct{})
	requests := make(chan struct{}, 3)
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests <- struct{}{}
		<-release
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"choices":[{"message":{"content":"{\"items\":[]}"},"finish_reason":"stop"}]}`))
	}))
	defer provider.Close()
	reader := vision.NewClient(provider.URL, "synthetic-test-key", "deepseek-flash")
	var picture bytes.Buffer
	if err := png.Encode(&picture, image.NewNRGBA(image.Rect(0, 0, 40, 20))); err != nil {
		t.Fatal(err)
	}
	background, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	finished := make(chan error, 2)
	for range 2 {
		go func() { _, err := reader.Extract(background, picture.Bytes(), "image/png", "test"); finished <- err }()
	}
	defer func() {
		close(release)
		for range 2 {
			<-finished
		}
	}()
	for range 2 {
		select {
		case <-requests:
		case <-time.After(3 * time.Second):
			t.Fatal("provider calls did not occupy both shared slots")
		}
	}
	handler := NewHandler(service, auth.NewService(service.pool, nil, nil, time.Hour), reader, NewWeatherClient(""))
	mux := http.NewServeMux()
	handler.Register(mux)
	requestContext, stop := context.WithTimeout(context.Background(), 3*time.Second)
	defer stop()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/day/import/preview?week_start=2026-11-09&timezone=UTC", bytes.NewReader(picture.Bytes())).WithContext(requestContext)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-AI-Consent", "deepseek")
	req.Header.Set("Content-Type", "image/png")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, req)
	var result struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if response.Code != http.StatusServiceUnavailable || response.Header().Get("Retry-After") != "5" || result.Error.Code != "import_busy" || len(requests) != 0 {
		t.Fatal("shared busy was mistaken for unreadable image or caused another provider call", response.Code, result.Error.Code, len(requests))
	}
	if len(handler.importSlots) != 0 {
		t.Fatal("shared busy did not release preparation slot")
	}
	var used int
	if err := service.pool.QueryRow(context.Background(), `SELECT used FROM day_import_limits WHERE user_id=1`).Scan(&used); err != nil || used != 1 {
		t.Fatal("valid prepared attempt should remain counted", used, err)
	}
}
