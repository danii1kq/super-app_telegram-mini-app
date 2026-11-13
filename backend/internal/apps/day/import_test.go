package day

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"image"
	"image/png"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/danii1kq/srunal/internal/platform/postgres"
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

func TestImportQuotaIsAtomicAndResetsByUTCDay(t *testing.T) {
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
	if _, err := pool.Exec(ctx, `CREATE TABLE users (id bigint PRIMARY KEY); INSERT INTO users VALUES (1)`); err != nil {
		t.Fatal(err)
	}
	migration, err := os.ReadFile("../../../../db/migrations/000013_day_import_limits.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, string(migration)); err != nil {
		t.Fatal(err)
	}
	service := NewService(&postgres.Pool{Pool: pool}, nil)
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
