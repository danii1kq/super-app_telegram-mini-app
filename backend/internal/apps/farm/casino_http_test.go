package farm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func exerciseCasinoHTTP(t *testing.T, pool *pgxpool.Pool, request func(string, string, string, int) *httptest.ResponseRecorder, readState func(*httptest.ResponseRecorder) State) {
	t.Helper()
	ctx := context.Background()
	for user := 1; user <= 2; user++ {
		s := initialState(time.Now())
		s.Coins = 10000
		normalizeState(&s)
		data, _ := json.Marshal(s)
		if _, err := pool.Exec(ctx, `UPDATE farm_players SET state=$2 WHERE user_id=$1`, user, data); err != nil {
			t.Fatal(err)
		}
	}
	create := func(user, n int) State {
		return readState(request("POST", "/api/v1/farm/actions", fmt.Sprintf(`{"kind":"flip_create","stake":1000,"side":"heads","request_id":%q}`, gameRequest(n)), user))
	}
	lobby := func() []Coinflip {
		w := request("GET", "/api/v1/farm/coinflips", "", 1)
		if w.Code != 200 {
			t.Fatalf("lobby HTTP%d %s", w.Code, w.Body)
		}
		var body struct {
			Flips []Coinflip `json:"flips"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		return body.Flips
	}
	created := create(1, 10)
	if created.Coins != 9000 {
		t.Fatal("host stake missing")
	}
	flips := lobby()
	if len(flips) != 1 || flips[0].HostID != "1" || flips[0].Status != "open" || flips[0].FeePercent != 8 {
		t.Fatal("real open match missing")
	}
	id := flips[0].ID
	act := func(kind string, n int) string {
		return fmt.Sprintf(`{"kind":%q,"item":%q,"request_id":%q}`, kind, id, gameRequest(n))
	}
	if request("POST", "/api/v1/farm/actions", act("flip_join", 11), 1).Code != 400 {
		t.Fatal("self join accepted")
	}
	if request("POST", "/api/v1/farm/actions", act("flip_cancel", 12), 2).Code != 400 {
		t.Fatal("opponent cancelled host stake")
	}
	parallel := func(kind string, user int) int {
		var wg sync.WaitGroup
		codes := make(chan int, 8)
		for i := 0; i < 8; i++ {
			wg.Add(1)
			go func(n int) {
				defer wg.Done()
				codes <- request("POST", "/api/v1/farm/actions", act(kind, 100+n), user).Code
			}(i)
		}
		wg.Wait()
		close(codes)
		ok := 0
		for code := range codes {
			if code == 200 {
				ok++
			} else if code != 400 {
				t.Fatalf("unexpected concurrent status %d", code)
			}
		}
		return ok
	}
	if parallel("flip_join", 2) != 1 {
		t.Fatal("coinflip accepted multiple guests")
	}
	a := readState(request("GET", "/api/v1/farm", "", 1))
	b := readState(request("GET", "/api/v1/farm", "", 2))
	if a.Coins+b.Coins != 19500 || len(a.GameHistory) != 1 || len(b.GameHistory) != 1 || a.GameHistory[0].Payout+b.GameHistory[0].Payout != 1500 || a.Sold != 0 || b.Sold != 0 {
		t.Fatal("coinflip minted money or credited harvest leaderboard")
	}
	flips = lobby()
	if flips[0].Status != "done" || flips[0].WinnerID == "" || flips[0].GuestID != "2" {
		t.Fatal("resolved match lost identities")
	}
	before := a.Coins
	create(1, 20)
	id = lobby()[0].ID
	if _, err := pool.Exec(ctx, `UPDATE farm_coinflips SET expires_at=now()-interval '1 second' WHERE id=$1`, id); err != nil {
		t.Fatal(err)
	}
	lobby()
	lobby()
	if readState(request("GET", "/api/v1/farm", "", 1)).Coins != before {
		t.Fatal("expiry did not refund exactly once")
	}
	create(1, 21)
	id = lobby()[0].ID
	if parallel("flip_cancel", 1) != 1 || readState(request("GET", "/api/v1/farm", "", 1)).Coins != before {
		t.Fatal("cancel duplicated or lost refund")
	}
	create(1, 22)
	id = lobby()[0].ID
	var race sync.WaitGroup
	raceCodes := make(chan int, 2)
	for _, entry := range []struct {
		kind    string
		user, n int
	}{{"flip_join", 2, 55}, {"flip_cancel", 1, 56}} {
		race.Add(1)
		go func(kind string, user, n int) {
			defer race.Done()
			raceCodes <- request("POST", "/api/v1/farm/actions", act(kind, n), user).Code
		}(entry.kind, entry.user, entry.n)
	}
	race.Wait()
	close(raceCodes)
	raceSuccess := 0
	for code := range raceCodes {
		if code == 200 {
			raceSuccess++
		} else if code != 400 {
			t.Fatalf("join/cancel race HTTP%d", code)
		}
	}
	if raceSuccess != 1 {
		t.Fatal("join and cancel both settled")
	}
	stateA := readState(request("GET", "/api/v1/farm", "", 1))
	stateB := readState(request("GET", "/api/v1/farm", "", 2))
	if total := stateA.Coins + stateB.Coins; total != 19500 && total != 19000 {
		t.Fatalf("join/cancel race minted or lost stake: %d", total)
	}
	crash := readState(request("POST", "/api/v1/farm/actions", fmt.Sprintf(`{"kind":"crash_start","stake":1000,"request_id":%q}`, gameRequest(30)), 1))
	id = crash.Crash.ID
	if strings.Contains(request("GET", "/api/v1/farm", "", 1).Body.String(), `"game_requests"`) {
		t.Fatal("private request ledger exposed")
	}
	if _, err := pool.Exec(ctx, `UPDATE farm_players SET state=jsonb_set(jsonb_set(jsonb_set(state,'{crash,bust_bp}','5000'),' {crash,status}','"running"'),'{crash,started_at}',to_jsonb(now()-interval '10 seconds')) WHERE user_id=1`); err != nil {
		t.Fatal(err)
	}
	before = readState(request("GET", "/api/v1/farm", "", 1)).Coins
	if parallel("crash_cashout", 1) != 1 {
		t.Fatal("crash paid more than once")
	}
	cashed := readState(request("GET", "/api/v1/farm", "", 1))
	if cashed.Crash.Status != "cashed" || cashed.Coins-before != cashed.Crash.Payout {
		t.Fatal("crash payout not persisted")
	}
	if _, err := pool.Exec(ctx, `UPDATE farm_players SET state=jsonb_set(state,'{barn}','[{"seed":"carrot","quality":"normal","mutation":"none","quantity":1,"value":32}]') WHERE user_id=1`); err != nil {
		t.Fatal(err)
	}
	quoteBody := `{"item":"legacy-0","seed":"lotus","upgrade":"crystal","stake":1000}`
	if w := request("POST", "/api/v1/farm/upgrade-quote", quoteBody, 1); w.Code != 200 {
		t.Fatalf("quote failed: %s", w.Body)
	}
	before = cashed.Coins
	var wg sync.WaitGroup
	codes := make(chan int, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			body := fmt.Sprintf(`{"kind":"plant_upgrade","item":"legacy-0","seed":"lotus","upgrade":"crystal","stake":1000,"request_id":%q}`, gameRequest(300+n))
			codes <- request("POST", "/api/v1/farm/actions", body, 1).Code
		}(i)
	}
	wg.Wait()
	close(codes)
	success := 0
	for code := range codes {
		if code == 200 {
			success++
		} else if code != 400 {
			t.Fatalf("upgrade HTTP%d", code)
		}
	}
	after := readState(request("GET", "/api/v1/farm", "", 1))
	if success != 1 || after.Coins != before-1000 {
		t.Fatal("one specimen upgraded more than once")
	}
	if request("GET", "/api/v1/farm/coinflips", "", 0).Code != 401 || request("POST", "/api/v1/farm/upgrade-quote", quoteBody, 0).Code != 401 {
		t.Fatal("unauthenticated game route accepted")
	}
	create(1, 700)
	id = lobby()[0].ID
	if _, err := pool.Exec(ctx, `UPDATE farm_coinflips SET fee_percent=10 WHERE id=$1`, id); err != nil {
		t.Fatal(err)
	}
	if lobby()[0].FeePercent != 10 {
		t.Fatal("legacy offer commission hidden")
	}
	readState(request("POST", "/api/v1/farm/actions", act("flip_join", 701), 2))
	legacyA := readState(request("GET", "/api/v1/farm", "", 1))
	legacyB := readState(request("GET", "/api/v1/farm", "", 2))
	if legacyA.GameHistory[len(legacyA.GameHistory)-1].Payout+legacyB.GameHistory[len(legacyB.GameHistory)-1].Payout != 1800 {
		t.Fatal("legacy offer lost its original 10% commission")
	}
}
