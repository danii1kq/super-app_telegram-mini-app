package farm

import (
	"context"
	"encoding/base64"
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
	"github.com/danii1kq/srunal/internal/platform/telegram"
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
	if _, err := pool.Exec(ctx, `CREATE TABLE users (id bigint PRIMARY KEY, public_name text NOT NULL, username text,telegram_id_lookup text, status text NOT NULL DEFAULT 'active');
	CREATE TABLE sessions (token_hash text PRIMARY KEY,user_id bigint,revoked_at timestamptz,expires_at timestamptz);
	INSERT INTO users (id,public_name) VALUES (1,'Alice'),(2,'Bob');`); err != nil {
		t.Fatal(err)
	}
	migration, err := os.ReadFile("../../../../db/migrations/000021_farm.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, string(migration)); err != nil {
		t.Fatal(err)
	}
	migration, err = os.ReadFile("../../../../db/migrations/000022_farm_themes.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, string(migration)); err != nil {
		t.Fatal(err)
	}
	migration, err = os.ReadFile("../../../../db/migrations/000023_farm_theme_worlds.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, string(migration)); err != nil {
		t.Fatal(err)
	}
	migration, err = os.ReadFile("../../../../db/migrations/000024_farm_coinflips.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, string(migration)); err != nil {
		t.Fatal(err)
	}
	migration, err = os.ReadFile("../../../../db/migrations/000025_farm_flip_fee.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, string(migration)); err != nil {
		t.Fatal(err)
	}
	for id := 1; id <= 2; id++ {
		if _, err := pool.Exec(ctx, `INSERT INTO sessions (token_hash,user_id,expires_at) VALUES ($1,$2,now()+interval '1 hour')`, secure.TokenHash(fmt.Sprintf("farm-test-%d", id)), id); err != nil {
			t.Fatal(err)
		}
	}
	db := &postgres.Pool{Pool: pool}
	mux := http.NewServeMux()
	index, err := secure.NewLookupIndex(base64.StdEncoding.EncodeToString(make([]byte, 32)))
	if err != nil {
		t.Fatal(err)
	}
	for id := 1; id <= 2; id++ {
		if _, err = pool.Exec(ctx, `UPDATE users SET telegram_id_lookup=$2 WHERE id=$1`, id, index.Of(strconv.Itoa(id*101))); err != nil {
			t.Fatal(err)
		}
	}
	bot := &fakePaymentBot{}
	handler := NewHandler(db, auth.NewService(db, nil, nil, time.Hour))
	handler.EnableStars(bot, index, "test-secret")
	if err = handler.SeedDevThemes(ctx); err != nil {
		t.Fatal(err)
	}
	handler.Register(mux)
	request := func(method, path, body string, user int) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		if user > 0 {
			r.Header.Set("Authorization", fmt.Sprintf("Bearer farm-test-%d", user))
			if path == "/bot/webhook" {
				r.Header.Set("X-Telegram-Bot-Api-Secret-Token", "test-secret")
			}
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
	upgraded := request("POST", "/api/v1/farm/actions", `{"kind":"upgrade","upgrade":"selection"}`, 1)
	var metadata struct {
		State    State     `json:"state"`
		Packs    []Pack    `json:"packs"`
		Upgrades []Upgrade `json:"upgrades"`
	}
	if err := json.Unmarshal(upgraded.Body.Bytes(), &metadata); err != nil {
		t.Fatal(err)
	}
	if upgraded.Code != 200 || metadata.State.Upgrades["selection"] != 1 || len(metadata.Upgrades) != 9 || len(metadata.Packs) != 4 {
		t.Fatalf("missing upgrade metadata: %s", upgraded.Body)
	}
	legend := 0
	for i, weight := range metadata.Packs[0].Weights {
		if Crops[i].Rarity == "legendary" {
			legend += weight
		}
	}
	if legend != 3 {
		t.Fatal("HTTP probabilities do not reflect purchased selection")
	}
	persisted := readState(request("GET", "/api/v1/farm", "", 1))
	if persisted.Upgrades["selection"] != 1 || persisted.Coins != packed.Coins-250 {
		t.Fatal("new upgrade did not persist")
	}
	bob := readState(request("GET", "/api/v1/farm", "", 2))
	if bob.Coins != 250 || bob.Plots[0] == nil || len(bob.Collection) != 0 {
		t.Fatal("player state leaked")
	}
	if request("POST", "/api/v1/farm/actions", `{"kind":"theme","theme":"lunar"}`, 1).Code != 400 {
		t.Fatal("locked theme was usable")
	}
	invoice := request("POST", "/api/v1/farm/themes/invoice", `{"theme":"lunar"}`, 1)
	if invoice.Code != 200 || bot.invoice.Currency != "XTR" || len(bot.invoice.Prices) != 1 || bot.invoice.Prices[0].Amount != 200 {
		t.Fatalf("invalid invoice: %s", invoice.Body)
	}
	checkoutBody := fmt.Sprintf(`{"pre_checkout_query":{"id":"test-q","from":{"id":101},"currency":"XTR","total_amount":200,"invoice_payload":%q}}`, bot.invoice.Payload)
	if request("POST", "/bot/webhook", checkoutBody, 0).Code != 401 {
		t.Fatal("unauthenticated payment webhook accepted")
	}
	if request("POST", "/bot/webhook", checkoutBody, 1).Code != 200 || !bot.approved {
		t.Fatal("valid precheckout denied")
	}
	if ownsTheme(readState(request("GET", "/api/v1/farm", "", 1)), "lunar") {
		t.Fatal("precheckout delivered unpaid theme")
	}
	paidBody := fmt.Sprintf(`{"message":{"from":{"id":101},"successful_payment":{"currency":"XTR","total_amount":200,"invoice_payload":%q,"telegram_payment_charge_id":"charge-1"}}}`, bot.invoice.Payload)
	wrongPayer := strings.Replace(paidBody, `"id":101`, `"id":202`, 1)
	if request("POST", "/bot/webhook", wrongPayer, 1).Code != 400 {
		t.Fatal("another user could claim invoice")
	}
	wrongAmount := strings.Replace(paidBody, `"total_amount":200`, `"total_amount":1`, 1)
	if request("POST", "/bot/webhook", wrongAmount, 1).Code != 400 {
		t.Fatal("wrong payment amount accepted")
	}
	for i := 0; i < 2; i++ {
		if w := request("POST", "/bot/webhook", paidBody, 1); w.Code != 200 {
			t.Fatalf("payment retry failed: %s", w.Body)
		}
	}
	paid := readState(request("GET", "/api/v1/farm", "", 1))
	if paid.Theme != "lunar" || len(paid.OwnedThemes) != 2 || paid.Coins != persisted.Coins {
		t.Fatal("paid theme not delivered exactly once")
	}
	redeemed := readState(request("POST", "/api/v1/farm/actions", `{"kind":"redeem","code":"  tropic  "}`, 1))
	if redeemed.Theme != "tropical" || len(redeemed.OwnedThemes) != 3 || redeemed.Coins != paid.Coins {
		t.Fatal("promo did not unlock cosmetic")
	}
	if request("POST", "/api/v1/farm/actions", `{"kind":"redeem","code":"TROPIC"}`, 1).Code != 400 {
		t.Fatal("code redeemed twice")
	}
	if request("POST", "/api/v1/farm/actions", `{"kind":"redeem","code":"NOT-A-CODE"}`, 1).Code != 400 {
		t.Fatal("unknown code accepted")
	}
	if readState(request("POST", "/api/v1/farm/actions", `{"kind":"theme","theme":"lunar"}`, 1)).Theme != "lunar" {
		t.Fatal("owned theme cannot be selected")
	}
	if ownsTheme(readState(request("GET", "/api/v1/farm", "", 2)), "lunar") {
		t.Fatal("theme purchase leaked to another player")
	}
	bobHarvest := readState(request("POST", "/api/v1/farm/actions", `{"kind":"harvest"}`, 2))
	specimen := bobHarvest.Barn[0]
	statuses = make(chan int, 6)
	for i := 0; i < 6; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			statuses <- request("POST", "/api/v1/farm/actions", fmt.Sprintf(`{"kind":"keep","item":%q}`, specimen.ID), 2).Code
		}()
	}
	wg.Wait()
	close(statuses)
	success = 0
	for status := range statuses {
		if status == 200 {
			success++
		} else if status != 400 {
			t.Fatalf("unexpected trophy status %d", status)
		}
	}
	protected := readState(request("GET", "/api/v1/farm", "", 2))
	if success != 1 || len(protected.Showcase) != 1 || len(protected.Barn) != 1 {
		t.Fatal("parallel keep duplicated specimen")
	}
	if len(readState(request("GET", "/api/v1/farm", "", 1)).Showcase) != 0 {
		t.Fatal("trophy leaked to another player")
	}
	breeding := readState(request("POST", "/api/v1/farm/actions", `{"kind":"breed","seed":"carrot"}`, 2))
	if breeding.Breeding == nil || breeding.Seeds["carrot"] != 2 {
		t.Fatal("breeding not saved")
	}
	if request("POST", "/api/v1/farm/actions", `{"kind":"claim_breed"}`, 2).Code != 400 {
		t.Fatal("early HTTP breeding claim")
	}
	// Advance only this test schema's fixture job, never live player clocks.
	if _, err = pool.Exec(ctx, `UPDATE farm_players SET state=jsonb_set(state,'{breeding,ready_at}',to_jsonb(now()-interval '1 second')) WHERE user_id=2`); err != nil {
		t.Fatal(err)
	}
	claimed := readState(request("POST", "/api/v1/farm/actions", `{"kind":"claim_breed"}`, 2))
	if claimed.Seeds["aurora_root"] != 1 || claimed.Breeding != nil || len(claimed.Hybrids) != 1 {
		t.Fatal("HTTP breeding claim not persisted")
	}
	if request("POST", "/api/v1/farm/actions", `{"kind":"claim_breed"}`, 2).Code != 400 {
		t.Fatal("double HTTP breeding claim")
	}
	if request("POST", "/api/v1/farm/actions", `{"kind":"start_expedition","item":"meadow"}`, 2).Code != 400 {
		t.Fatal("locked route accepted before level 6")
	}
	if _, err = pool.Exec(ctx, `UPDATE farm_players SET state=jsonb_set(state,'{xp}','1000'::jsonb) WHERE user_id=2`); err != nil {
		t.Fatal(err)
	}
	started := readState(request("POST", "/api/v1/farm/actions", `{"kind":"start_expedition","item":"meadow"}`, 2))
	if started.Expedition == nil || started.Coins != claimed.Coins-90 {
		t.Fatal("expedition did not persist its price and route")
	}
	if request("POST", "/api/v1/farm/actions", `{"kind":"claim_expedition"}`, 2).Code != 400 {
		t.Fatal("early expedition claim accepted")
	}
	if _, err = pool.Exec(ctx, `UPDATE farm_players SET state=jsonb_set(state,'{expedition,ready_at}',to_jsonb(now()-interval '1 second')) WHERE user_id=2`); err != nil {
		t.Fatal(err)
	}
	statuses = make(chan int, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			statuses <- request("POST", "/api/v1/farm/actions", `{"kind":"claim_expedition"}`, 2).Code
		}()
	}
	wg.Wait()
	close(statuses)
	success = 0
	for status := range statuses {
		if status == 200 {
			success++
		} else if status != 400 {
			t.Fatalf("unexpected expedition claim status %d", status)
		}
	}
	returned := readState(request("GET", "/api/v1/farm", "", 2))
	if success != 1 || returned.Expedition != nil || returned.Seeds["pepper"] != started.Seeds["pepper"]+2 || returned.Seeds["lavender"] != started.Seeds["lavender"]+1 || returned.XP != started.XP+10 || returned.Coins != started.Coins {
		t.Fatal("concurrent expedition claims duplicated or lost rewards")
	}
	if readState(request("GET", "/api/v1/farm", "", 1)).Expedition != nil {
		t.Fatal("expedition leaked to another account")
	}

	// Test an order against actual daily metadata, then verify its shared weekly ledger.
	order := returned.Orders.Orders[0]
	if order.Seed != "carrot" && order.Seed != "radish" && order.Seed != "cucumber" {
		t.Fatal("expected an easy order")
	}
	fixture := []Harvest{{ID: "order-fixture", Seed: order.Seed, Quality: "normal", Mutation: "none", Quantity: order.Quantity, Value: 32 * order.Quantity}}
	fixtureJSON, err := json.Marshal(fixture)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `UPDATE farm_players SET state=jsonb_set(state,'{barn}',$1::jsonb) WHERE user_id=2`, fixtureJSON); err != nil {
		t.Fatal(err)
	}
	beforeOrder := readState(request("GET", "/api/v1/farm", "", 2))
	fulfilled := readState(request("POST", "/api/v1/farm/actions", fmt.Sprintf(`{"kind":"fulfill_order","item":%q}`, order.ID), 2))
	if fulfilled.Sold-beforeOrder.Sold != order.Coins || fulfilled.Coins-beforeOrder.Coins != order.Coins {
		t.Fatal("order sale not counted")
	}
	var weekly int
	if err = pool.QueryRow(ctx, `SELECT revenue FROM farm_weekly_sales WHERE week=$1 AND user_id=2`, weekStart(time.Now().UTC())).Scan(&weekly); err != nil {
		t.Fatal(err)
	}
	if weekly != fulfilled.Sold {
		t.Fatal("order sale missing from weekly leaderboard")
	}
	exerciseCasinoHTTP(t, pool, request, readState)
}

type fakePaymentBot struct {
	invoice  telegram.InvoiceRequest
	approved bool
}

func (b *fakePaymentBot) CreateInvoiceLink(_ context.Context, invoice telegram.InvoiceRequest) (string, error) {
	b.invoice = invoice
	return "https://t.me/$test-invoice", nil
}
func (b *fakePaymentBot) AnswerPreCheckout(_ context.Context, _ string, ok bool, _ string) error {
	b.approved = ok
	return nil
}
