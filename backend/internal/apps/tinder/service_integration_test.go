package tinder

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
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
	"github.com/jackc/pgx/v5/pgxpool"
)

func tinderFixture(t *testing.T) (*Service, *http.ServeMux) {
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
	schema := fmt.Sprintf("tinder_test_%d", time.Now().UnixNano())
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
	migrations, err := os.ReadDir("../../../../db/migrations")
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range migrations {
		if !strings.HasSuffix(file.Name(), ".up.sql") {
			continue
		}
		raw, err := os.ReadFile("../../../../db/migrations/" + file.Name())
		if err != nil {
			t.Fatal(err)
		}
		if _, err = pool.Exec(ctx, string(raw)); err != nil {
			t.Fatal(file.Name(), err)
		}
	}
	for id := 1; id <= 4; id++ {
		if _, err = pool.Exec(ctx, `INSERT INTO users(id,telegram_id_lookup,telegram_id_sealed,key_id,public_name,username)OVERRIDING SYSTEM VALUE VALUES($1,$2,''::bytea,'test',$2,$3)`, id, fmt.Sprint(id), fmt.Sprintf("test_user%d", id)); err != nil {
			t.Fatal(err)
		}
		if _, err = pool.Exec(ctx, `INSERT INTO user_settings(user_id)VALUES($1)`, id); err != nil {
			t.Fatal(err)
		}
		if _, err = pool.Exec(ctx, `INSERT INTO sessions(token_hash,user_id,expires_at)VALUES($1,$2,now()+interval '1 hour')`, secure.TokenHash(fmt.Sprintf("tinder-%d", id)), id); err != nil {
			t.Fatal(err)
		}
	}
	cipher, err := secure.NewCipher(base64.StdEncoding.EncodeToString(make([]byte, 32)))
	if err != nil {
		t.Fatal(err)
	}
	db := &postgres.Pool{Pool: pool}
	service := NewService(db, cipher)
	service.SetDelivery(cipher)
	mux := http.NewServeMux()
	NewHandler(service, auth.NewService(db, nil, cipher, time.Hour)).Register(mux)
	return service, mux
}

func TestTinderConcurrentMatchPhotosAndBlocks(t *testing.T) {
	s, mux := tinderFixture(t)
	ctx := context.Background()
	var photo bytes.Buffer
	if err := png.Encode(&photo, image.NewRGBA(image.Rect(0, 0, 16, 24))); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SavePhoto(ctx, 4, photo.Bytes()); err == nil {
		t.Fatal("silently lost a photo before profile creation")
	}
	for _, id := range []int64{1, 2, 3} {
		if _, err := s.SaveProfile(ctx, id, "Private bio", "", "University", "", nil, 21, true); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := s.SavePhoto(ctx, 1, photo.Bytes()); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for _, pair := range [][2]int64{{1, 2}, {2, 1}} {
		wg.Add(1)
		go func(pair [2]int64) {
			defer wg.Done()
			if _, _, err := s.Like(ctx, pair[0], pair[1]); err != nil {
				t.Error(err)
			}
		}(pair)
	}
	wg.Wait()
	matches, err := s.Matches(ctx, 1)
	if err != nil || len(matches) != 1 || matches[0].Username != "test_user2" {
		t.Fatal(matches, err)
	}
	if _, _, err = s.Like(ctx, 2, 1); err != nil {
		t.Fatal(err)
	}
	var count int
	if err = s.pool.QueryRow(ctx, `SELECT count(*) FROM notification_delivery WHERE kind='tinder.matched'`).Scan(&count); err != nil || count != 2 {
		t.Fatal("duplicate or missing match notifications", count, err)
	}
	if _, err = s.pool.Exec(ctx, `INSERT INTO social_blocks(blocker_id,blocked_id)VALUES(1,2),(1,3)`); err != nil {
		t.Fatal(err)
	}
	if _, _, err = s.Like(ctx, 3, 1); err == nil {
		t.Fatal("blocked user could like profile")
	}
	matches, err = s.Matches(ctx, 1)
	if err != nil || len(matches) != 0 {
		t.Fatal("blocked match visible", matches, err)
	}
	if _, _, err = s.Photo(ctx, 2, 1); err == nil {
		t.Fatal("blocked photo visible")
	}
	for _, request := range []struct {
		token  string
		status int
	}{{"", 401}, {"tinder-2", 404}, {"tinder-1", 200}} {
		r := httptest.NewRequest("GET", "/api/v1/tinder/photo/1", nil)
		if request.token != "" {
			r.Header.Set("Authorization", "Bearer "+request.token)
		}
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		if w.Code != request.status {
			t.Fatal(w.Code, w.Body.String())
		}
	}
	var raw []byte
	if err = s.pool.QueryRow(ctx, `SELECT bio_sealed FROM tinder_profiles WHERE user_id=1`).Scan(&raw); err != nil || bytes.Contains(raw, []byte("Private bio")) {
		t.Fatal("bio is not encrypted", err)
	}
}
