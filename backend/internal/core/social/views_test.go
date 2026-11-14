package social

import (
	"encoding/json"
	"strconv"
	"strings"
	"sync"
	"testing"
)

func TestTelegramProfileLink(t *testing.T) {
	for _, name := range []string{"", "bad/name", "name?x=1", "https://evil.test", "abcd", "name\nvalue"} {
		if telegramURL(name) != nil {
			t.Fatalf("unsafe Telegram link for %q", name)
		}
	}
	if link := telegramURL("Safe_User"); link == nil || *link != "https://t.me/Safe_User" {
		t.Fatal("valid public username rejected")
	}
}

func TestUniqueViewsAndProfiles(t *testing.T) {
	s, ctx := socialTestService(t)
	var author, viewer, stranger int64
	for n, id := range []*int64{&author, &viewer, &stranger} {
		if err := s.pool.QueryRow(ctx, `INSERT INTO users (telegram_id_lookup,telegram_id_sealed,key_id,public_name,username)
   VALUES ($1,''::bytea,'test',$1,$2) RETURNING id`, "view_person"+strconv.Itoa(n), "Public_user"+strconv.Itoa(n)).Scan(id); err != nil {
			t.Fatal(err)
		}
	}
	public, err := s.CreatePost(ctx, author, NewPost{Body: "Public", Audience: "public"})
	if err != nil {
		t.Fatal(err)
	}
	private, err := s.CreatePost(ctx, author, NewPost{Body: "For followers", Audience: "followers"})
	if err != nil {
		t.Fatal(err)
	}
	if count, err := s.RecordView(ctx, author, public.ID); err != nil || count != 0 {
		t.Fatalf("author counted: %d %v", count, err)
	}
	var wait sync.WaitGroup
	for n := 0; n < 8; n++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			count, err := s.RecordView(ctx, viewer, public.ID)
			if err != nil || count != 1 {
				t.Errorf("repeated concurrent view: %d %v", count, err)
			}
		}()
	}
	wait.Wait()
	if count, err := s.RecordView(ctx, stranger, public.ID); err != nil || count != 2 {
		t.Fatalf("second viewer: %d %v", count, err)
	}
	_, err = s.RecordView(ctx, stranger, private.ID)
	requireHidden(t, err)
	if err := s.Follow(ctx, viewer, author); err != nil {
		t.Fatal(err)
	}
	if err := s.Follow(ctx, author, viewer); err != nil {
		t.Fatal(err)
	}
	if count, err := s.RecordView(ctx, viewer, private.ID); err != nil || count != 1 {
		t.Fatalf("allowed private view: %d %v", count, err)
	}
	if _, err := s.ToggleReaction(ctx, viewer, public.ID, "like"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateComment(ctx, viewer, public.ID, "A comment"); err != nil {
		t.Fatal(err)
	}
	detail, err := s.byID(ctx, public.ID, viewer)
	if err != nil || detail.Views != 2 || detail.Likes != 1 || detail.Comments != 1 || !detail.LikedByMe {
		t.Fatalf("detail counters: %#v %v", detail, err)
	}
	profile, err := s.Profile(ctx, viewer, author)
	if err != nil || !profile.Following || !profile.FollowsYou || profile.Followers != 1 || profile.FollowingCount != 1 || profile.PostsCount != 2 {
		t.Fatalf("profile: %#v %v", profile, err)
	}
	dto := toProfileDTO(profile)
	if !dto.IsFriend || dto.TelegramURL == nil {
		t.Fatal("mutual following must show friendship and public contact")
	}
	raw, err := json.Marshal(dto)
	if err != nil || strings.Contains(string(raw), "telegram_id") || strings.Contains(string(raw), "sealed") {
		t.Fatalf("private identity exposed: %s %v", raw, err)
	}
	posts, err := s.AuthorPosts(ctx, viewer, author, 0, 20)
	if err != nil || len(posts) != 2 {
		t.Fatalf("followed author posts: %d %v", len(posts), err)
	}
	if err := s.Unfollow(ctx, viewer, author); err != nil {
		t.Fatal(err)
	}
	profile, err = s.Profile(ctx, viewer, author)
	if err != nil || profile.Following || !profile.FollowsYou || profile.PostsCount != 1 {
		t.Fatalf("unfollow profile: %#v %v", profile, err)
	}
	posts, err = s.AuthorPosts(ctx, viewer, author, 0, 20)
	if err != nil || len(posts) != 1 || posts[0].ID != public.ID {
		t.Fatalf("restricted author posts exposed: %#v %v", posts, err)
	}
	_, err = s.RecordView(ctx, viewer, private.ID)
	requireHidden(t, err)
	if _, err := s.pool.Exec(ctx, `INSERT INTO social_blocks (blocker_id,blocked_id) VALUES ($1,$2)`, viewer, author); err != nil {
		t.Fatal(err)
	}
	_, err = s.Profile(ctx, viewer, author)
	requireHidden(t, err)
	_, err = s.RecordView(ctx, viewer, public.ID)
	requireHidden(t, err)
	_, err = s.AuthorPosts(ctx, viewer, author, 0, 20)
	requireHidden(t, err)
	requireHidden(t, s.Follow(ctx, author, viewer))
	requireHidden(t, s.Follow(ctx, viewer, author))
	if err := s.DeletePost(ctx, author, public.ID); err != nil {
		t.Fatal(err)
	}
	_, err = s.RecordView(ctx, stranger, public.ID)
	requireHidden(t, err)

	const largeID int64 = 9007199254740993
	if _, err := s.pool.Exec(ctx, `INSERT INTO users (id,telegram_id_lookup,telegram_id_sealed,key_id,public_name)
	 OVERRIDING SYSTEM VALUE VALUES ($1,'large-user',''::bytea,'test','Large user')`, largeID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreatePost(ctx, largeID, NewPost{Body: "Large author", Audience: "public"}); err != nil {
		t.Fatal(err)
	}
	posts, err = s.AuthorPosts(ctx, stranger, largeID, largeID, 20)
	if err != nil || len(posts) != 1 {
		t.Fatalf("bigint profile cursor: %d %v", len(posts), err)
	}
}
