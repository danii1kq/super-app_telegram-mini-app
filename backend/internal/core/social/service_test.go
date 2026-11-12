package social

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

func TestPostAndCommentValidation(t *testing.T) {
	s := NewService(nil, nil)
	for _, input := range []NewPost{
		{Body: " \n", Audience: "public"},
		{Body: strings.Repeat("я", 4001), Audience: "public"},
		{Body: "text", Audience: "unknown"},
		{Body: "text", Audience: "community"},
		{Body: "text", Audience: "public", Community: new(int64)},
	} {
		if _, err := s.CreatePost(context.Background(), 1, input); err == nil {
			t.Fatalf("accepted invalid post: %#v", input)
		}
	}
	for _, body := range []string{" \n", strings.Repeat("я", 2001)} {
		if _, err := s.CreateComment(context.Background(), 1, 1, body); err == nil {
			t.Fatal("accepted invalid comment")
		}
	}
	for _, raw := range []string{"-1", "0", "abc", "9223372036854775808"} {
		if _, err := cursorID(raw); err == nil {
			t.Fatalf("accepted invalid cursor %q", raw)
		}
	}
	if value, err := cursorID(""); err != nil || value != 0 {
		t.Fatal("empty cursor must start the first page")
	}
	if value, err := cursorID("123"); err != nil || value != 123 {
		t.Fatal("valid cursor rejected")
	}
}

func TestFeedCommentsAndReactionsAccess(t *testing.T) {
	s, ctx := socialTestService(t)
	var author, follower, stranger, blocked int64
	for index, id := range []*int64{&author, &follower, &stranger, &blocked} {
		if err := s.pool.QueryRow(ctx, `INSERT INTO users (telegram_id_lookup, telegram_id_sealed, key_id, public_name)
		 VALUES ($1, ''::bytea, 'test', $1) RETURNING id`, "person"+strconv.Itoa(index)).Scan(id); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.Follow(ctx, follower, author); err != nil {
		t.Fatal(err)
	}
	public, err := s.CreatePost(ctx, author, NewPost{Body: " Public ", Audience: "public"})
	if err != nil {
		t.Fatal(err)
	}
	private, err := s.CreatePost(ctx, author, NewPost{Body: "Private", Audience: "followers"})
	if err != nil {
		t.Fatal(err)
	}
	comment, err := s.CreateComment(ctx, follower, private.ID, " Secret comment ")
	if err != nil || comment.Body != "Secret comment" {
		t.Fatalf("create private comment: %#v %v", comment, err)
	}
	var publicBody *string
	var ciphertext []byte
	if err := s.pool.QueryRow(ctx, `SELECT body_public, body_sealed FROM social_comments WHERE id = $1`, comment.ID).Scan(&publicBody, &ciphertext); err != nil {
		t.Fatal(err)
	}
	if publicBody != nil || len(ciphertext) == 0 {
		t.Fatal("private comment must only be stored encrypted")
	}
	comments, err := s.Comments(ctx, author, private.ID, 0)
	if err != nil || len(comments) != 1 || comments[0].Body != "Secret comment" {
		t.Fatalf("authorized comments: %#v %v", comments, err)
	}
	_, err = s.Comments(ctx, stranger, private.ID, 0)
	requireHidden(t, err)
	_, err = s.CreateComment(ctx, stranger, private.ID, "intrusion")
	requireHidden(t, err)
	_, err = s.ToggleReaction(ctx, stranger, private.ID, "like")
	requireHidden(t, err)
	if _, err := s.pool.Exec(ctx, `INSERT INTO social_blocks (blocker_id, blocked_id) VALUES ($1, $2)`, blocked, author); err != nil {
		t.Fatal(err)
	}
	_, err = s.Comments(ctx, blocked, public.ID, 0)
	requireHidden(t, err)
	_, err = s.ToggleReaction(ctx, blocked, public.ID, "like")
	requireHidden(t, err)
	if posts, err := s.Feed(ctx, blocked, 0, 20); err != nil || len(posts) != 0 {
		t.Fatalf("blocked author's posts leaked: %#v %v", posts, err)
	}
	if added, err := s.ToggleReaction(ctx, follower, public.ID, "like"); err != nil || !added {
		t.Fatalf("like: %v %v", added, err)
	}
	if added, err := s.ToggleReaction(ctx, follower, public.ID, "like"); err != nil || added {
		t.Fatalf("unlike: %v %v", added, err)
	}
	if err := s.DeletePost(ctx, author, private.ID); err != nil {
		t.Fatal(err)
	}
	_, err = s.CreateComment(ctx, follower, private.ID, "deleted")
	requireHidden(t, err)

	var community int64
	if err := s.pool.QueryRow(ctx, `INSERT INTO social_communities (owner_id, name_sealed, key_id) VALUES ($1, ''::bytea, 'test') RETURNING id`, author).Scan(&community); err != nil {
		t.Fatal(err)
	}
	if _, err := s.pool.Exec(ctx, `INSERT INTO social_members (community_id, user_id) VALUES ($1, $2)`, community, author); err != nil {
		t.Fatal(err)
	}
	closed, err := s.CreatePost(ctx, author, NewPost{Body: "Community", Audience: "community", Community: &community})
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.Comments(ctx, follower, closed.ID, 0)
	requireHidden(t, err)
	if _, err := s.pool.Exec(ctx, `UPDATE social_members SET state = 'left' WHERE community_id = $1`, community); err != nil {
		t.Fatal(err)
	}
	_, err = s.Comments(ctx, author, closed.ID, 0)
	requireHidden(t, err)

	for i := 0; i < 24; i++ {
		if _, err := s.CreatePost(ctx, author, NewPost{Body: "Page", Audience: "public"}); err != nil {
			t.Fatal(err)
		}
	}
	first, err := s.Feed(ctx, follower, 0, 20)
	if err != nil || len(first) != 20 {
		t.Fatalf("first page: %d %v", len(first), err)
	}
	second, err := s.Feed(ctx, follower, first[19].ID, 20)
	if err != nil || len(second) != 5 {
		t.Fatalf("second page: %d %v", len(second), err)
	}
	seen := map[int64]bool{}
	for _, post := range append(first, second...) {
		if seen[post.ID] {
			t.Fatal("duplicate post across pages")
		}
		seen[post.ID] = true
	}
	for i := 0; i < 23; i++ {
		if _, err := s.CreateComment(ctx, follower, public.ID, "Page"); err != nil {
			t.Fatal(err)
		}
	}
	firstComments, err := s.Comments(ctx, follower, public.ID, 0)
	if err != nil || len(firstComments) != 21 {
		t.Fatalf("first comments page: %d %v", len(firstComments), err)
	}
	secondComments, err := s.Comments(ctx, follower, public.ID, firstComments[19].ID)
	if err != nil || len(secondComments) != 3 {
		t.Fatalf("second comments page: %d %v", len(secondComments), err)
	}
	if firstComments[19].ID <= secondComments[0].ID {
		t.Fatal("comments pagination overlaps")
	}
}

func requireHidden(t *testing.T, err error) {
	t.Helper()
	var apiErr *httpapi.Error
	if !errors.As(err, &apiErr) || apiErr.Status != http.StatusNotFound {
		t.Fatalf("wanted neutral not found, got %v", err)
	}
}

func socialTestService(t *testing.T) (*Service, context.Context) {
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
	schema := "social_test_" + strconv.FormatInt(time.Now().UnixNano(), 36)
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
	for _, file := range []string{"000001_core_identity.up.sql", "000002_social_chat.up.sql"} {
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
