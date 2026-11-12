package chat

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/danii1kq/srunal/internal/core/auth"
	"github.com/danii1kq/srunal/internal/platform/httpapi"
	"github.com/danii1kq/srunal/internal/platform/postgres"
	"github.com/danii1kq/srunal/internal/platform/secure"
)

func testChatService(t *testing.T) *Service {
	t.Helper()
	dsn := os.Getenv("SRUNAL_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set SRUNAL_TEST_DATABASE_URL to run isolated PostgreSQL chat tests")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	schema := pgx.Identifier{"chat_test_" + strconv.FormatInt(time.Now().UnixNano(), 10)}.Sanitize()
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		admin.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = admin.Exec(context.Background(), "DROP SCHEMA "+schema+" CASCADE")
		admin.Close()
	})
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
	if _, err := pool.Exec(ctx, `
		CREATE TABLE users (id bigint PRIMARY KEY, public_name text NOT NULL, username text, status text NOT NULL DEFAULT 'active');
		CREATE TABLE sessions (token_hash text PRIMARY KEY, user_id bigint, revoked_at timestamptz, expires_at timestamptz);
		CREATE TABLE social_communities (id bigint PRIMARY KEY);
		CREATE TABLE social_blocks (blocker_id bigint, blocked_id bigint);
		INSERT INTO users (id, public_name) VALUES (1, 'Alice'), (2, 'Bob'), (3, 'Eve'), (4, 'Charlie');
	`); err != nil {
		t.Fatal(err)
	}
	migration, err := os.ReadFile("../../../../db/migrations/000003_chat.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, string(migration)); err != nil {
		t.Fatal(err)
	}
	cipher, err := secure.NewCipher(base64.StdEncoding.EncodeToString(make([]byte, 32)))
	if err != nil {
		t.Fatal(err)
	}
	return NewService(&postgres.Pool{Pool: pool}, cipher)
}

func TestHistoryHTTPAndPrivateStream(t *testing.T) {
	s := testChatService(t)
	ctx := context.Background()
	conversation, err := s.OpenDirect(ctx, 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	for seq := 1; seq <= 3; seq++ {
		if _, err := s.Send(ctx, 2, conversation.ID, fmt.Sprintf("http-%d", seq), "Private"); err != nil {
			t.Fatal(err)
		}
	}
	for userID, token := range map[int64]string{1: "alice-test-token", 3: "eve-test-token"} {
		if _, err := s.pool.Exec(ctx, `INSERT INTO sessions (token_hash, user_id, expires_at) VALUES ($1, $2, now() + interval '1 hour')`, secure.TokenHash(token), userID); err != nil {
			t.Fatal(err)
		}
	}
	authService := auth.NewService(s.pool, nil, nil, time.Hour)
	mux := http.NewServeMux()
	NewHandler(s, authService, NewHub()).Register(mux)
	request := func(path, token string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		recorder := httptest.NewRecorder()
		mux.ServeHTTP(recorder, req)
		return recorder
	}
	response := request(fmt.Sprintf("/api/v1/conversations/%d/messages?limit=2", conversation.ID), "alice-test-token")
	var page struct {
		Messages []struct {
			Seq string `json:"seq"`
		} `json:"messages"`
		HasMore bool `json:"has_more"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &page); err != nil || response.Code != http.StatusOK || len(page.Messages) != 2 || page.Messages[0].Seq != "2" || !page.HasMore {
		t.Fatalf("wrong latest HTTP page: %s, status %d, err %v", response.Body.String(), response.Code, err)
	}
	for _, path := range []string{
		fmt.Sprintf("/api/v1/conversations/%d/messages", conversation.ID),
		fmt.Sprintf("/api/v1/events/stream?conversation=%d", conversation.ID),
	} {
		if response := request(path, "eve-test-token"); response.Code != http.StatusNotFound {
			t.Fatalf("private endpoint %s returned %d: %s", path, response.Code, response.Body.String())
		}
	}
	if response := request("/api/v1/events/stream?conversation=invalid", "alice-test-token"); response.Code != http.StatusBadRequest {
		t.Fatalf("invalid stream cursor returned %d", response.Code)
	}
}

func assertChatStatus(t *testing.T, err error, want int) {
	t.Helper()
	var apiErr *httpapi.Error
	if !errors.As(err, &apiErr) || apiErr.Status != want {
		t.Fatalf("wanted status %d, got %v", want, err)
	}
}

func TestChatHistoryPreviewAndAccess(t *testing.T) {
	s := testChatService(t)
	ctx := context.Background()
	conversation, err := s.OpenDirect(ctx, 1, 2)
	if err != nil || conversation.Title != "Bob" || conversation.PartnerID != 2 {
		t.Fatalf("open direct: %+v, %v", conversation, err)
	}
	var first Message
	for seq := int64(1); seq <= 120; seq++ {
		sender := int64(1)
		if seq > 100 {
			sender = 2
		}
		message, err := s.Send(ctx, sender, conversation.ID, fmt.Sprintf("message-%d", seq), fmt.Sprintf("Message %d", seq))
		if err != nil {
			t.Fatal(err)
		}
		if seq == 1 {
			first = message
		}
	}
	assertPage := func(query HistoryQuery, start, end int64, more bool) {
		t.Helper()
		messages, hasMore, err := s.History(ctx, 1, conversation.ID, query)
		if err != nil || len(messages) != int(end-start+1) || hasMore != more {
			t.Fatalf("page: %d messages, more=%v, err=%v", len(messages), hasMore, err)
		}
		for index, message := range messages {
			if message.Seq != start+int64(index) || message.Text != fmt.Sprintf("Message %d", message.Seq) {
				t.Fatalf("wrong page message: %+v", message)
			}
		}
	}
	assertPage(HistoryQuery{}, 71, 120, true)
	before := int64(71)
	assertPage(HistoryQuery{Before: &before, Limit: 50}, 21, 70, true)
	before = 21
	assertPage(HistoryQuery{Before: &before, Limit: 50}, 1, 20, false)
	after := int64(0)
	assertPage(HistoryQuery{After: &after, Limit: 100}, 1, 100, true)
	after = 100
	assertPage(HistoryQuery{After: &after, Limit: 100}, 101, 120, false)
	_, _, err = s.History(ctx, 3, conversation.ID, HistoryQuery{Limit: 50})
	assertChatStatus(t, err, 404)
	assertChatStatus(t, s.MarkRead(ctx, 3, conversation.ID, 120), 404)
	_, err = s.Send(ctx, 3, conversation.ID, "stranger", "private access")
	assertChatStatus(t, err, 404)

	replay, err := s.Send(ctx, 1, conversation.ID, "message-1", "Message 1")
	if err != nil || replay.ID != first.ID || !replay.CreatedAt.Equal(first.CreatedAt) || replay.SenderName != "Alice" {
		t.Fatalf("replay lost original metadata: %+v, %v", replay, err)
	}
	_, err = s.Send(ctx, 1, conversation.ID, "message-1", "changed")
	assertChatStatus(t, err, 409)
	other, err := s.OpenDirect(ctx, 1, 4)
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.Send(ctx, 1, other.ID, "message-1", "Message 1")
	assertChatStatus(t, err, 409)
	if _, err := s.Send(ctx, 4, other.ID, "recent", "Most recent"); err != nil {
		t.Fatal(err)
	}
	list, err := s.List(ctx, 1)
	if err != nil || len(list) != 2 || list[0].ID != other.ID || list[0].Title != "Charlie" {
		t.Fatalf("list should sort by message time, not seq: %+v, %v", list, err)
	}
	if list[0].LastMessage == nil || list[0].LastMessage.Text != "Most recent" || list[1].Unread != 20 {
		t.Fatalf("incorrect preview/unread: %+v", list)
	}
	stranger, err := s.List(ctx, 3)
	if err != nil || len(stranger) != 0 {
		t.Fatalf("stranger can list private conversation: %+v, %v", stranger, err)
	}
	if err := s.MarkRead(ctx, 1, conversation.ID, 9223372036854775807); err != nil {
		t.Fatal(err)
	}
	if err := s.MarkRead(ctx, 1, conversation.ID, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Send(ctx, 2, conversation.ID, "unread-after-clamp", "New unread"); err != nil {
		t.Fatal(err)
	}
	list, err = s.List(ctx, 1)
	if err != nil || list[0].ID != conversation.ID || list[0].Unread != 1 || list[0].LastRead != 120 {
		t.Fatalf("future read marker hid incoming message: %+v, %v", list, err)
	}
	if _, err := s.pool.Exec(ctx, `INSERT INTO social_blocks VALUES (1, 2)`); err != nil {
		t.Fatal(err)
	}
	_, err = s.Send(ctx, 2, conversation.ID, "blocked-send", "Should not send")
	assertChatStatus(t, err, 404)
}

func TestOpenDirectConcurrent(t *testing.T) {
	s := testChatService(t)
	ctx := context.Background()
	results := make(chan Conversation, 8)
	errors := make(chan error, 8)
	var group sync.WaitGroup
	for index := 0; index < 8; index++ {
		group.Add(1)
		go func(index int) {
			defer group.Done()
			viewer, other := int64(1), int64(2)
			if index%2 != 0 {
				viewer, other = other, viewer
			}
			conversation, err := s.OpenDirect(ctx, viewer, other)
			results <- conversation
			errors <- err
		}(index)
	}
	group.Wait()
	close(results)
	close(errors)
	for err := range errors {
		if err != nil {
			t.Fatal(err)
		}
	}
	var id int64
	for conversation := range results {
		if id == 0 {
			id = conversation.ID
		}
		if conversation.ID != id {
			t.Fatalf("concurrent opens created different dialogs: %d, %d", id, conversation.ID)
		}
	}
}
