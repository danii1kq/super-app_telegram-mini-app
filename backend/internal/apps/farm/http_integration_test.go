package farm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/danii1kq/srunal/internal/core/auth"
	"github.com/danii1kq/srunal/internal/platform/postgres"
	"github.com/danii1kq/srunal/internal/platform/secure"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestFarmHTTPPersistenceAndConcurrentSales(t *testing.T) {
	dsn := os.Getenv("SRUNAL_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set SRUNAL_TEST_DATABASE_URL for isolated PostgreSQL tests")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	schema := pgx.Identifier{"farm_test_" + strconv.FormatInt(time.Now().UnixNano(), 10)}.Sanitize()
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		admin.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = admin.Exec(context.Background(), "DROP SCHEMA "+schema+" CASCADE"); admin.Close() })
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
	if _, err := pool.Exec(ctx, `CREATE TABLE users (id bigint PRIMARY KEY, public_name text NOT NULL, username text, status text NOT NULL DEFAULT 'active');
	CREATE TABLE sessions (token_hash text PRIMARY KEY,user_id bigint,revoked_at timestamptz,expires_at timestamptz);
	INSERT INTO users (id,public_name) VALUES (1,'Alice'),(2,'Bob');`); err != nil {
		t.Fatal(err)
	}
	migration, err := os.ReadFile("../../../../db/migrations/000011_farm.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, string(migration)); err != nil {
		t.Fatal(err)
	}
	for id := 1; id <= 2; id++ {
		if _, err := pool.Exec(ctx, `INSERT INTO sessions (token_hash,user_id,expires_at) VALUES ($1,$2,now()+interval '1 hour')`, secure.TokenHash(fmt.Sprintf("farm-test-%d", id)), id); err != nil {
			t.Fatal(err)
		}
	}
	db := &postgres.Pool{Pool: pool}
	mux := http.NewServeMux()
	NewHandler(db, auth.NewService(db, nil, nil, time.Hour)).Register(mux)
	request := func(method, path, body string, user int) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		if user > 0 {
			r.Header.Set("Authorization", fmt.Sprintf("Bearer farm-test-%d", user))
		}
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		return w
	}
	readState := func(w *httptest.ResponseRecorder) State {
		t.Helper()
		if w.Code != 200 {
			t.Fatalf("HTTP %d: %s", w.Code, w.Body)
		}
		var response struct {
			State State `json:"state"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatal(err)
		}
		return response.State
	}
	if got := request("GET", "/api/v1/farm", "", 0).Code; got != 401 {
		t.Fatalf("unauthenticated access: %d", got)
	}
	first := readState(request("GET", "/api/v1/farm", "", 1))
	if first.Coins != 250 || len(first.Plots) != 4 {
		t.Fatal("missing starter state")
	}
	harvested := readState(request("POST", "/api/v1/farm/actions", `{"kind":"harvest"}`, 1))
	expected := 0
	for _, item := range harvested.Barn {
		expected += item.Value
	}
	var wg sync.WaitGroup
	statuses := make(chan int, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			statuses <- request("POST", "/api/v1/farm/actions", `{"kind":"sell"}`, 1).Code
		}()
	}
	wg.Wait()
	close(statuses)
	success := 0
	for code := range statuses {
		if code == 200 {
			success++
		} else if code != 400 {
			t.Fatalf("unexpected sale status: %d", code)
		}
	}
	if success != 1 {
		t.Fatalf("double sale: %d accepted", success)
	}
	sold := readState(request("GET", "/api/v1/farm", "", 1))
	if sold.Coins != first.Coins+expected || len(sold.Barn) != 0 {
		t.Fatal("sale was not atomic or persisted")
	}
	if len(sold.Collection) != 2 {
		t.Fatal("sale erased collection")
	}
	board := request("GET", "/api/v1/farm/leaderboard", "", 1)
	var rankingResponse struct {
		Entries []ranking `json:"entries"`
	}
	if err := json.Unmarshal(board.Body.Bytes(), &rankingResponse); err != nil {
		t.Fatal(err)
	}
	if board.Code != 200 || len(rankingResponse.Entries) != 1 || rankingResponse.Entries[0].Revenue != int64(expected) || !rankingResponse.Entries[0].Mine {
		t.Fatalf("wrong leaderboard: %s", board.Body)
	}
	planted := readState(request("POST", "/api/v1/farm/actions", `{"kind":"plant","slot":0,"seed":"carrot","care":"water"}`, 1))
	if planted.Plots[0] == nil || planted.Plots[0].ReadyAt.Sub(planted.Plots[0].PlantedAt) != GrowTime {
		t.Fatal("planting duration corrupted")
	}
	if code := request("POST", "/api/v1/farm/actions", `{"kind":"harvest"}`, 1).Code; code != 400 {
		t.Fatal("harvest before maturity")
	}
	if code := request("POST", "/api/v1/farm/actions", `{"kind":"buy","seed":"carrot","coins":999999}`, 1).Code; code != 400 {
		t.Fatal("accepted client currency")
	}
	packed := readState(request("POST", "/api/v1/farm/actions", `{"kind":"pack","pack":"garden"}`, 1))
	if packed.Coins != planted.Coins-75 || packed.Opened != 1 {
		t.Fatal("pack purchase did not persist")
	}
	total := func(s State) int {
		n := 0
		for _, v := range s.Seeds {
			n += v
		}
		return n
	}
	if total(packed) != total(planted)+3 {
		t.Fatal("pack must contain three seeds")
	}
	bob := readState(request("GET", "/api/v1/farm", "", 2))
	if bob.Coins != 250 || bob.Plots[0] == nil || len(bob.Collection) != 0 {
		t.Fatal("player state leaked")
	}
}
