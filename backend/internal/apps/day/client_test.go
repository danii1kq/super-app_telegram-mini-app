package day

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func testInitData(id string) string { return url.Values{"user": {`{"id":` + id + `}`}}.Encode() }

func TestActualSruDayContractAndPrivateCreation(t *testing.T) {
	var authCalls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api/v1/auth/telegram" {
			authCalls++
			if r.Header.Get("Authorization") != "" {
				t.Error("deployment token was used as user credentials")
			}
			var body map[string]string
			if json.NewDecoder(r.Body).Decode(&body) != nil || body["init_data"] != testInitData("42") {
				t.Error("missing verified payload")
			}
			json.NewEncoder(w).Encode(map[string]any{"token": "upstream-user-token", "expires_at": time.Now().Add(time.Hour), "user": map[string]int{"id": 7, "telegram_id": 42}})
			return
		}
		if r.Header.Get("Authorization") != "Bearer upstream-user-token" || r.Header.Get("X-Telegram-Init-Data") != "" {
			t.Error("data call must use upstream user session only")
		}
		if r.Method == http.MethodGet {
			if r.URL.Query().Get("from") != "2026-11-09T00:00:00Z" || r.URL.Query().Get("to") != "2026-11-16T00:00:00Z" {
				t.Error("wrong interval")
			}
			w.Write([]byte(`{"occurrences":[{"event_id":123,"title":"Coffee","starts_at":"2026-11-10T09:00:00Z","ends_at":"2026-11-10T10:00:00Z","all_day":false,"location":"Cafe"},{"event_id":123,"title":"Coffee","starts_at":"2026-11-11T09:00:00Z","ends_at":"2026-11-11T10:00:00Z","all_day":false}]}`))
			return
		}
		var body map[string]any
		if json.NewDecoder(r.Body).Decode(&body) != nil {
			t.Error("invalid create JSON")
		}
		if body["visibility"] != "private" || body["source_ref"] != nil || body["group_id"] != nil {
			t.Error("creation expanded privacy or claimed upstream idempotency")
		}
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id":124,"title":"Meeting","starts_at":"2026-11-10T09:00:00Z","ends_at":"2026-11-10T10:00:00Z","all_day":false}`))
	}))
	defer server.Close()
	client := NewClient(server.URL, "must-not-be-forwarded")
	from := time.Date(2026, 11, 9, 0, 0, 0, 0, time.UTC)
	events, err := client.ListEvents(context.Background(), testInitData("42"), from, from.AddDate(0, 0, 7))
	if err != nil || len(events) != 2 || events[0].ID != "123" || events[1].ID != "123" || events[0].Location != "Cafe" {
		t.Fatalf("actual occurrences: %#v %v", events, err)
	}
	event, err := client.CreateEvent(context.Background(), testInitData("42"), CreateEventInput{Title: "Meeting", StartsAt: "2026-11-10T09:00:00Z", EndsAt: "2026-11-10T10:00:00Z", SourceRef: "pulse:123"})
	if err != nil || event.ID != "124" {
		t.Fatalf("create: %#v %v", event, err)
	}
	if authCalls != 1 {
		t.Fatalf("expected one short-lived upstream session, got %d", authCalls)
	}
}

func TestRejectDifferentIdentityAndBrokenUpstream(t *testing.T) {
	for _, body := range []string{`{"token":"secret","expires_at":"2099-01-01T00:00:00Z","user":{"id":1,"telegram_id":99}}`, `not json`} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(body)) }))
		_, err := NewClient(server.URL, "").authenticate(context.Background(), testInitData("42"))
		server.Close()
		if err == nil {
			t.Fatal("accepted mismatched identity or malformed response")
		}
	}
	for _, code := range []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusInternalServerError} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(code)
			w.Write([]byte("sensitive upstream diagnostics"))
		}))
		_, err := NewClient(server.URL, "").authenticate(context.Background(), testInitData("42"))
		server.Close()
		if err == nil || strings.Contains(err.Error(), "sensitive") {
			t.Fatal("upstream error leaked or was ignored")
		}
	}
}

func TestAuthenticationDoesNotFollowRedirects(t *testing.T) {
	leaked := false
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { leaked = true }))
	defer target.Close()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusTemporaryRedirect)
	}))
	defer server.Close()
	if _, err := NewClient(server.URL, "").authenticate(context.Background(), testInitData("42")); err == nil || leaked {
		t.Fatal("redirect forwarded authentication data")
	}
}

func TestEventValidationAndAllDayDeadlineDST(t *testing.T) {
	valid := CreateEventInput{Title: " Meeting ", StartsAt: "2026-11-10T09:00:00Z", EndsAt: "2026-11-10T10:00:00Z"}
	if err := validateEvent(&valid); err != nil || valid.Title != "Meeting" {
		t.Fatal("valid event rejected")
	}
	for _, input := range []CreateEventInput{{Title: " ", StartsAt: valid.StartsAt, EndsAt: valid.EndsAt}, {Title: "Meeting", StartsAt: valid.StartsAt, EndsAt: valid.StartsAt}, {Title: "Meeting", StartsAt: "bad", EndsAt: valid.EndsAt}, {Title: "Meeting", StartsAt: valid.StartsAt, EndsAt: "2027-01-10T10:00:00Z"}} {
		if validateEvent(&input) == nil {
			t.Fatal("accepted invalid event")
		}
	}
	deadline, err := deadlineDay(CreateEventInput{AllDay: true, StartsAt: "2026-03-08T14:00:00Z", EndsAt: "2026-03-08T14:00:00Z"}, "America/New_York")
	if err != nil {
		t.Fatal(err)
	}
	start, _ := time.Parse(time.RFC3339, deadline.StartsAt)
	end, _ := time.Parse(time.RFC3339, deadline.EndsAt)
	if start.Hour() != 0 || end.Hour() != 0 || end.Sub(start) != 23*time.Hour {
		t.Fatalf("deadline must use local calendar day across DST: %#v", deadline)
	}
}
