package chat

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/danii1kq/srunal/internal/core/auth"
	"github.com/danii1kq/srunal/internal/platform/secure"
)

func groupTestKey(value int) string { return fmt.Sprintf("00000000-0000-4000-8000-%012d", value) }

func TestGroupValidation(t *testing.T) {
	for _, raw := range []string{"", "0", "-1", "+2", "02", " 2", "2.0", "9223372036854775808"} {
		if _, err := positiveID(raw); err == nil {
			t.Errorf("accepted non-canonical ID %q", raw)
		}
	}
	if id, err := positiveID("9007199254740993"); err != nil || id != 9007199254740993 {
		t.Fatalf("lost bigint ID: %d, %v", id, err)
	}
	ids, err := groupMemberIDs(1, []int64{1, 2, 2})
	if err != nil || len(ids) != 2 || ids[0] != 1 || ids[1] != 2 {
		t.Fatalf("group deduplication: %v, %v", ids, err)
	}
	for _, selected := range [][]int64{nil, {1, 1}, {0, 2}, {-1, 2}} {
		if _, err := groupMemberIDs(1, selected); err == nil {
			t.Errorf("accepted invalid group members %v", selected)
		}
	}
	s := NewService(nil, nil)
	for _, clientID := range []string{"", "group", "00000000-0000-4000-8000-00000000000Z"} {
		if _, err := s.CreateGroup(context.Background(), 1, clientID, "Group", []int64{2}); err == nil {
			t.Errorf("accepted invalid group request key %q", clientID)
		}
	}
	for _, title := range []string{" \n", strings.Repeat("я", 101)} {
		if _, err := s.CreateGroup(context.Background(), 1, groupTestKey(1), title, []int64{2}); err == nil {
			t.Errorf("accepted invalid title %q", title)
		}
	}
}

func TestGroupMembersAccessAndEncryptedTitle(t *testing.T) {
	s := testChatService(t)
	ctx := context.Background()
	group, err := s.CreateGroup(ctx, 1, groupTestKey(1), " Private group ", []int64{1, 2, 2})
	if err != nil || group.Kind != "group" || group.Title != "Private group" {
		t.Fatalf("create group: %+v, %v", group, err)
	}
	var sealed []byte
	var keyID string
	if err := s.pool.QueryRow(ctx, `SELECT title_sealed, key_id FROM chat_conversations WHERE id = $1`, group.ID).Scan(&sealed, &keyID); err != nil {
		t.Fatal(err)
	}
	if keyID == "" || len(sealed) == 0 || bytes.Contains(sealed, []byte(group.Title)) {
		t.Fatal("group title is not encrypted")
	}
	if _, err := s.content.OpenString(secure.Sealed{KeyID: keyID, Ciphertext: sealed}, titleAAD(group.ID+1)); err == nil {
		t.Fatal("group title decrypted in another conversation context")
	}
	members, err := s.GroupMembers(ctx, 2, group.ID)
	if err != nil || len(members) != 2 || members[0].UserID != 1 || members[0].Role != "owner" {
		t.Fatalf("members: %+v, %v", members, err)
	}
	_, err = s.GroupMembers(ctx, 3, group.ID)
	assertChatStatus(t, err, 404)
	assertChatStatus(t, s.AddGroupMember(ctx, 3, group.ID, 4), 404)
	assertChatStatus(t, s.RemoveGroupMember(ctx, 3, group.ID, 2), 404)
	assertChatStatus(t, s.AddGroupMember(ctx, 2, group.ID, 3), 403)
	assertChatStatus(t, s.RemoveGroupMember(ctx, 2, group.ID, 1), 403)
	assertChatStatus(t, s.RemoveGroupMember(ctx, 1, group.ID, 1), 409)

	if _, err := s.Send(ctx, 1, group.ID, "group-first", "Private message"); err != nil {
		t.Fatal(err)
	}
	if err := s.AddGroupMember(ctx, 1, group.ID, 3); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Send(ctx, 1, group.ID, "group-second", "Another message"); err != nil {
		t.Fatal(err)
	}
	if err := s.AddGroupMember(ctx, 1, group.ID, 3); err != nil {
		t.Fatal(err)
	}
	var read int64
	if err := s.pool.QueryRow(ctx, `SELECT last_read_seq FROM chat_members WHERE conversation_id = $1 AND user_id = 3`, group.ID).Scan(&read); err != nil || read != 1 {
		t.Fatalf("join/repeated add read marker: %d, %v", read, err)
	}
	if err := s.RemoveGroupMember(ctx, 1, group.ID, 2); err != nil {
		t.Fatal(err)
	}
	_, _, err = s.History(ctx, 2, group.ID, HistoryQuery{})
	assertChatStatus(t, err, 404)
	assertChatStatus(t, s.MarkRead(ctx, 2, group.ID, 2), 404)
	_, err = s.Send(ctx, 2, group.ID, "removed-send", "Hidden")
	assertChatStatus(t, err, 404)
	_, err = s.GroupMembers(ctx, 2, group.ID)
	assertChatStatus(t, err, 404)
	if list, err := s.List(ctx, 2); err != nil || len(list) != 0 {
		t.Fatalf("removed participant still sees group: %+v, %v", list, err)
	}
	if err := s.AddGroupMember(ctx, 1, group.ID, 2); err != nil {
		t.Fatal(err)
	}
	if err := s.pool.QueryRow(ctx, `SELECT last_read_seq FROM chat_members WHERE conversation_id = $1 AND user_id = 2`, group.ID).Scan(&read); err != nil || read != 2 {
		t.Fatalf("rejoined read marker: %d, %v", read, err)
	}
	if err := s.RemoveGroupMember(ctx, 2, group.ID, 2); err != nil {
		t.Fatal(err)
	}
	if err := s.RemoveGroupMember(ctx, 3, group.ID, 3); err != nil {
		t.Fatal(err)
	}
	if err := s.RemoveGroupMember(ctx, 1, group.ID, 1); err != nil {
		t.Fatal(err)
	}
	if list, err := s.List(ctx, 1); err != nil || len(list) != 0 {
		t.Fatalf("left group still listed: %+v, %v", list, err)
	}
}

func TestGroupBlocksAndConcurrentCapacity(t *testing.T) {
	s := testChatService(t)
	ctx := context.Background()
	if _, err := s.pool.Exec(ctx, `INSERT INTO social_blocks VALUES (2, 3)`); err != nil {
		t.Fatal(err)
	}
	_, err := s.CreateGroup(ctx, 1, groupTestKey(1), "Blocked pair", []int64{2, 3})
	assertChatStatus(t, err, 404)
	group, err := s.CreateGroup(ctx, 1, groupTestKey(2), "Allowed pair", []int64{2})
	if err != nil {
		t.Fatal(err)
	}
	assertChatStatus(t, s.AddGroupMember(ctx, 1, group.ID, 3), 404)
	if _, err := s.pool.Exec(ctx, `UPDATE users SET status = 'disabled' WHERE id = 4`); err != nil {
		t.Fatal(err)
	}
	assertChatStatus(t, s.AddGroupMember(ctx, 1, group.ID, 4), 404)
	_, err = s.CreateGroup(ctx, 1, groupTestKey(3), "Inactive member", []int64{4})
	assertChatStatus(t, err, 404)
	if _, err := s.pool.Exec(ctx, `DELETE FROM social_blocks; UPDATE users SET status = 'active';
	 INSERT INTO users (id, public_name) SELECT n, 'User ' || n FROM generate_series(5, 52) n`); err != nil {
		t.Fatal(err)
	}
	selected := make([]int64, 48)
	for index := range selected {
		selected[index] = int64(index + 2)
	}
	full, err := s.CreateGroup(ctx, 1, groupTestKey(4), "Capacity", selected)
	if err != nil {
		t.Fatal(err)
	}
	var workers sync.WaitGroup
	results := make(chan error, 2)
	for _, id := range []int64{50, 51} {
		workers.Add(1)
		go func(id int64) {
			defer workers.Done()
			results <- s.AddGroupMember(ctx, 1, full.ID, id)
		}(id)
	}
	workers.Wait()
	close(results)
	added := 0
	for err := range results {
		if err == nil {
			added++
		} else {
			assertChatStatus(t, err, 409)
		}
	}
	members, err := s.GroupMembers(ctx, 1, full.ID)
	if err != nil || len(members) != 50 || added != 1 {
		t.Fatalf("concurrent adds exceeded capacity: members=%d, added=%d, err=%v", len(members), added, err)
	}
	selected = append(selected, 50, 51)
	if _, err := s.CreateGroup(ctx, 1, groupTestKey(5), "Too many", selected); err == nil {
		t.Fatal("created group with 51 participants")
	}
}

func TestGroupHTTP(t *testing.T) {
	s := testChatService(t)
	ctx := context.Background()
	for user, token := range map[int64]string{1: "group-owner", 2: "group-member", 3: "group-outsider"} {
		if _, err := s.pool.Exec(ctx, `INSERT INTO sessions (token_hash, user_id, expires_at) VALUES ($1, $2, now() + interval '1 hour')`, secure.TokenHash(token), user); err != nil {
			t.Fatal(err)
		}
	}
	mux := http.NewServeMux()
	NewHandler(s, auth.NewService(s.pool, nil, nil, time.Hour), NewHub()).registerGroups(mux)
	request := func(method, path, token, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		return rec
	}
	response := request(http.MethodPost, "/api/v1/conversations/groups", "group-owner", `{"client_group_id":"00000000-0000-4000-8000-000000000001","title":" Friends ","user_ids":["2","2"]}`)
	var created struct {
		ID    string `json:"id"`
		Title string `json:"title"`
		Kind  string `json:"kind"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &created); err != nil || response.Code != 201 || created.Title != "Friends" || created.Kind != "group" {
		t.Fatalf("create HTTP: %d %s, err=%v", response.Code, response.Body.String(), err)
	}
	replayed := request(http.MethodPost, "/api/v1/conversations/groups", "group-owner", `{"client_group_id":"00000000-0000-4000-8000-000000000001","title":"Friends","user_ids":["2"]}`)
	var replay struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(replayed.Body.Bytes(), &replay); err != nil || replayed.Code != 201 || replay.ID != created.ID {
		t.Fatalf("lost-response retry created a duplicate: %d %s, %v", replayed.Code, replayed.Body.String(), err)
	}
	if response := request(http.MethodPost, "/api/v1/conversations/groups", "group-owner", `{"title":"No key","user_ids":["2"]}`); response.Code != 400 {
		t.Fatalf("missing request key accepted: %d", response.Code)
	}
	path := "/api/v1/conversations/" + created.ID + "/members"
	for _, test := range []struct {
		method, path, token, body string
		status                    int
	}{
		{http.MethodGet, path, "group-member", "", 200},
		{http.MethodGet, path, "group-outsider", "", 404},
		{http.MethodPost, path, "group-member", `{"user_id":"3"}`, 403},
		{http.MethodPost, path, "group-owner", `{"user_id":"03"}`, 400},
		{http.MethodPost, path, "group-owner", `{"user_id":"3"}`, 200},
		{http.MethodDelete, path + "/1", "group-owner", "", 409},
		{http.MethodDelete, path + "/2", "group-owner", "", 200},
		{http.MethodGet, path, "group-member", "", 404},
	} {
		response := request(test.method, test.path, test.token, test.body)
		if response.Code != test.status {
			t.Errorf("%s %s: %d, want %d: %s", test.method, test.path, response.Code, test.status, response.Body.String())
		}
	}
	direct, err := s.OpenDirect(ctx, 1, 4)
	if err != nil {
		t.Fatal(err)
	}
	if response := request(http.MethodGet, fmt.Sprintf("/api/v1/conversations/%s/members", strconv.FormatInt(direct.ID, 10)), "group-owner", ""); response.Code != 404 {
		t.Fatalf("direct conversation exposed group API: %d", response.Code)
	}
}

func TestGroupCreateIdempotency(t *testing.T) {
	s := testChatService(t)
	ctx := context.Background()
	var workers sync.WaitGroup
	results := make(chan Conversation, 8)
	failures := make(chan error, 8)
	for n := 0; n < 8; n++ {
		workers.Add(1)
		go func(n int) {
			defer workers.Done()
			ids := []int64{2, 3}
			if n%2 == 0 {
				ids = []int64{3, 2, 2, 1}
			}
			group, err := s.CreateGroup(ctx, 1, groupTestKey(1), "Shared group", ids)
			results <- group
			failures <- err
		}(n)
	}
	workers.Wait()
	close(results)
	close(failures)
	for err := range failures {
		if err != nil {
			t.Fatal(err)
		}
	}
	var id int64
	for group := range results {
		if id == 0 {
			id = group.ID
		}
		if group.ID != id {
			t.Fatalf("concurrent retries created duplicate groups: %d, %d", id, group.ID)
		}
	}
	var count int
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM chat_conversations`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("duplicate group rows: %d, %v", count, err)
	}
	_, err := s.CreateGroup(ctx, 1, groupTestKey(1), "Changed title", []int64{2, 3})
	assertChatStatus(t, err, 409)
	_, err = s.CreateGroup(ctx, 1, groupTestKey(1), "Shared group", []int64{2, 4})
	assertChatStatus(t, err, 409)
	otherOwner, err := s.CreateGroup(ctx, 2, groupTestKey(1), "Shared group", []int64{1, 3})
	if err != nil || otherOwner.ID == id {
		t.Fatalf("request key was not scoped to owner: %+v, %v", otherOwner, err)
	}
	if err := s.AddGroupMember(ctx, 1, id, 4); err != nil {
		t.Fatal(err)
	}
	_, err = s.CreateGroup(ctx, 1, groupTestKey(1), "Shared group", []int64{2, 3})
	assertChatStatus(t, err, 409)
	members, err := s.GroupMembers(ctx, 1, id)
	if err != nil || len(members) != 4 {
		t.Fatalf("replay reset later membership changes: %+v, %v", members, err)
	}
	for _, userID := range []int64{2, 3, 4, 1} {
		if err := s.RemoveGroupMember(ctx, 1, id, userID); err != nil {
			t.Fatal(err)
		}
	}
	_, err = s.CreateGroup(ctx, 1, groupTestKey(1), "Shared group", []int64{2, 3})
	assertChatStatus(t, err, 404)
}
