package pulse

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/danii1kq/srunal/internal/platform/httpapi"
	"github.com/danii1kq/srunal/internal/platform/postgres"
	"github.com/danii1kq/srunal/internal/platform/secure"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestCreateValidation(t *testing.T) {
	s := NewService(nil, nil)
	valid := Audience{Users: []int64{2}}
	for _, input := range []struct {
		text     string
		end      time.Time
		audience Audience
	}{
		{" \n ", time.Now().Add(time.Hour), valid},
		{strings.Repeat("я", 281), time.Now().Add(time.Hour), valid},
		{"text", time.Now(), valid},
		{"text", time.Now().Add(25 * time.Hour), valid},
		{"text", time.Now().Add(time.Hour), Audience{}},
		{"text", time.Now().Add(time.Hour), Audience{Users: []int64{1}}},
		{"text", time.Now().Add(time.Hour), Audience{Users: []int64{-1}}},
		{"text", time.Now().Add(time.Hour), Audience{Users: make([]int64, 101)}},
		{"text", time.Now().Add(time.Hour), Audience{Community: new(int64)}},
	} {
		if _, err := s.Create(context.Background(), 1, input.text, input.end, input.audience); err == nil {
			t.Fatalf("accepted invalid input: %#v", input)
		}
	}
}

func TestPulseAudienceResponsesAndExpiry(t *testing.T) {
	s, ctx := pulseTestService(t)
	var author, recipient, outsider int64
	for index, id := range []*int64{&author, &recipient, &outsider} {
		if err := s.pool.QueryRow(ctx, `INSERT INTO users (telegram_id_lookup, telegram_id_sealed, key_id, public_name)
		 VALUES ($1, ''::bytea, 'test', $1) RETURNING id`, "person"+strconv.Itoa(index)).Scan(id); err != nil {
			t.Fatal(err)
		}
	}
	status, err := s.Create(ctx, author, " Coffee? ", time.Now().Add(time.Hour), Audience{Users: []int64{author, recipient, recipient}})
	if err != nil || status.Text != "Coffee?" {
		t.Fatalf("create: %#v %v", status, err)
	}
	var sealed []byte
	var audienceCount int
	if err := s.pool.QueryRow(ctx, `SELECT text_sealed FROM pulse_statuses WHERE id = $1`, status.ID).Scan(&sealed); err != nil || strings.Contains(string(sealed), "Coffee?") {
		t.Fatalf("plaintext stored: %v", err)
	}
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM pulse_audience WHERE status_id = $1`, status.ID).Scan(&audienceCount); err != nil || audienceCount != 1 {
		t.Fatalf("duplicate/self audience: %d %v", audienceCount, err)
	}
	feed, err := s.Feed(ctx, recipient)
	if err != nil || len(feed) != 1 || feed[0].Text != "Coffee?" {
		t.Fatalf("recipient feed: %#v %v", feed, err)
	}
	if feed, err := s.Feed(ctx, outsider); err != nil || len(feed) != 0 {
		t.Fatalf("outsider feed: %#v %v", feed, err)
	}
	requirePulseError(t, s.Respond(ctx, outsider, status.ID), http.StatusNotFound)
	for range 2 {
		if err := s.Respond(ctx, recipient, status.ID); err != nil {
			t.Fatal(err)
		}
	}
	people, err := s.Responders(ctx, author, status.ID)
	if err != nil || len(people) != 1 || people[0].ID != recipient {
		t.Fatalf("responders: %#v %v", people, err)
	}
	_, err = s.Responders(ctx, recipient, status.ID)
	requirePulseError(t, err, http.StatusNotFound)
	for range 2 {
		if err := s.CancelResponse(ctx, recipient, status.ID); err != nil {
			t.Fatal(err)
		}
	}
	people, err = s.Responders(ctx, author, status.ID)
	if err != nil || len(people) != 0 {
		t.Fatalf("cancel response: %#v %v", people, err)
	}

	// A valid status must remain usable during its final partial minute.
	if _, err := s.pool.Exec(ctx, `UPDATE pulse_statuses SET expires_at = now() + interval '30 seconds' WHERE id = $1`, status.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.Respond(ctx, recipient, status.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(ctx, author, status.ID); err != nil {
		t.Fatal(err)
	}
	requirePulseError(t, s.Respond(ctx, recipient, status.ID), http.StatusConflict)
	requirePulseError(t, s.Respond(ctx, outsider, status.ID), http.StatusNotFound)
	if feed, err := s.Feed(ctx, recipient); err != nil || len(feed) != 0 {
		t.Fatalf("closed feed: %#v %v", feed, err)
	}

	active, err := s.Create(ctx, author, "Still active", time.Now().Add(time.Hour), Audience{Users: []int64{recipient}})
	if err != nil {
		t.Fatal(err)
	}
	for _, pair := range [][2]int64{{recipient, author}, {author, recipient}} {
		if _, err := s.pool.Exec(ctx, `INSERT INTO social_blocks (blocker_id, blocked_id) VALUES ($1, $2)`, pair[0], pair[1]); err != nil {
			t.Fatal(err)
		}
		_, err := s.Create(ctx, author, "Hidden", time.Now().Add(time.Hour), Audience{Users: []int64{recipient}})
		requirePulseError(t, err, http.StatusUnprocessableEntity)
		requirePulseError(t, s.Respond(ctx, recipient, status.ID), http.StatusNotFound)
		requirePulseError(t, s.Respond(ctx, recipient, active.ID), http.StatusNotFound)
		if feed, err := s.Feed(ctx, recipient); err != nil || len(feed) != 0 {
			t.Fatalf("blocked feed: %#v %v", feed, err)
		}
		if _, err := s.pool.Exec(ctx, `DELETE FROM social_blocks`); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := s.pool.Exec(ctx, `UPDATE pulse_statuses SET created_at = now() - interval '2 hours', expires_at = now() - interval '1 hour' WHERE id = $1`, active.ID); err != nil {
		t.Fatal(err)
	}
	requirePulseError(t, s.Respond(ctx, recipient, active.ID), http.StatusConflict)
	if feed, err := s.Feed(ctx, recipient); err != nil || len(feed) != 0 {
		t.Fatalf("expired feed: %#v %v", feed, err)
	}
	_, err = s.Create(ctx, author, "Bad recipient", time.Now().Add(time.Hour), Audience{Users: []int64{999999}})
	requirePulseError(t, err, http.StatusUnprocessableEntity)

	var community int64
	if err := s.pool.QueryRow(ctx, `INSERT INTO social_communities (owner_id, name_sealed, key_id)
	 VALUES ($1, ''::bytea, 'test') RETURNING id`, author).Scan(&community); err != nil {
		t.Fatal(err)
	}
	for _, id := range []int64{author, recipient} {
		if _, err := s.pool.Exec(ctx, `INSERT INTO social_members (community_id, user_id, state) VALUES ($1, $2, 'active')`, community, id); err != nil {
			t.Fatal(err)
		}
	}
	group, err := s.Create(ctx, author, "Group coffee", time.Now().Add(time.Hour), Audience{Community: &community})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Respond(ctx, recipient, group.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.pool.Exec(ctx, `UPDATE social_members SET state = 'left' WHERE community_id = $1 AND user_id = $2`, community, recipient); err != nil {
		t.Fatal(err)
	}
	requirePulseError(t, s.Respond(ctx, recipient, group.ID), http.StatusNotFound)
	if _, err := s.pool.Exec(ctx, `UPDATE social_members SET state = 'active' WHERE community_id = $1`, community); err != nil {
		t.Fatal(err)
	}
	if _, err := s.pool.Exec(ctx, `UPDATE social_communities SET state = 'archived' WHERE id = $1`, community); err != nil {
		t.Fatal(err)
	}
	requirePulseError(t, s.Respond(ctx, recipient, group.ID), http.StatusNotFound)
	_, err = s.Create(ctx, author, "Closed group", time.Now().Add(time.Hour), Audience{Community: &community})
	requirePulseError(t, err, http.StatusForbidden)
}

func requirePulseError(t *testing.T, err error, status int) {
	t.Helper()
	var apiErr *httpapi.Error
	if !errors.As(err, &apiErr) || apiErr.Status != status {
		t.Fatalf("wanted HTTP %d, got %v", status, err)
	}
}

func pulseTestService(t *testing.T) (*Service, context.Context) {
	t.Helper()
	dsn := os.Getenv("SRUNAL_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set SRUNAL_TEST_DATABASE_URL for isolated PostgreSQL integration tests")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	admin, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	schema := "pulse_test_" + strconv.FormatInt(time.Now().UnixNano(), 36)
	name := pgx.Identifier{schema}.Sanitize()
	if _, err := admin.Exec(ctx, `CREATE SCHEMA `+name); err != nil {
		admin.Close(ctx)
		t.Fatal(err)
	}
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		pool.Close()
		_, _ = admin.Exec(context.Background(), `DROP SCHEMA `+name+` CASCADE`)
		_ = admin.Close(context.Background())
	})
	for _, file := range []string{"000001_core_identity.up.sql", "000002_social_chat.up.sql", "000007_pulse.up.sql"} {
		migration, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "db", "migrations", file))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, string(migration)); err != nil {
			t.Fatal(err)
		}
	}
	cipher, err := secure.NewCipher(base64.StdEncoding.EncodeToString(make([]byte, 32)))
	if err != nil {
		t.Fatal(err)
	}
	return NewService(&postgres.Pool{Pool: pool}, cipher), ctx
}
