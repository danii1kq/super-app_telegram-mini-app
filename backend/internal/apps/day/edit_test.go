package day

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/danii1kq/srunal/internal/platform/httpapi"
)

func TestOrdinaryEditsCheckOwnerSeriesAndVersion(t *testing.T) {
	var mode atomic.Int32
	var patches atomic.Int32
	var race atomic.Bool
	record := func() map[string]any {
		out := map[string]any{"id": 12, "owner_id": 7, "title": "Before", "starts_at": "2026-11-13T10:00:00Z", "ends_at": "2026-11-13T11:00:00Z", "all_day": false, "location": "Room", "version": 3, "visibility": "private", "source": "manual"}
		switch mode.Load() {
		case 1:
			out["owner_id"] = 8
		case 2:
			out["group_id"] = 4
		case 3:
			out["rrule"] = "FREQ=WEEKLY"
		case 4:
			out["source"] = "moodle"
		case 5:
			out["visibility"] = "public"
		}
		return out
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/auth/telegram" {
			json.NewEncoder(w).Encode(map[string]any{"token": "private-upstream-token", "expires_at": time.Now().Add(time.Hour), "user": map[string]int{"id": 7, "telegram_id": 42}})
			return
		}
		if r.URL.Path != "/api/v1/events/12" || r.Header.Get("Authorization") != "Bearer private-upstream-token" {
			t.Error("wrong event or authorization")
		}
		if r.Method == http.MethodGet {
			json.NewEncoder(w).Encode(record())
			return
		}
		patches.Add(1)
		if r.Method != http.MethodPatch || r.Header.Get("If-Match") != "3" {
			t.Error("PATCH must forward exact loaded version")
		}
		var body map[string]any
		if json.NewDecoder(r.Body).Decode(&body) != nil {
			t.Error("bad patch JSON")
		}
		for _, key := range []string{"owner_id", "group_id", "rrule", "source", "visibility", "version"} {
			if _, ok := body[key]; ok {
				t.Errorf("edit changed protected metadata %s", key)
			}
		}
		if body["location"] != " " || body["all_day"] != true {
			t.Error("explicit clear or all-day value lost")
		}
		if race.Load() {
			w.WriteHeader(http.StatusConflict)
			w.Write([]byte("sensitive upstream context"))
			return
		}
		out := record()
		out["version"] = 4
		out["title"] = body["title"]
		out["location"] = ""
		out["all_day"] = true
		json.NewEncoder(w).Encode(out)
	}))
	defer server.Close()
	client := NewClient(server.URL, "")
	input := CreateEventInput{Title: "After", StartsAt: "2026-11-13T10:00:00Z", EndsAt: "2026-11-13T10:01:00Z", AllDay: true}
	event, err := client.GetEvent(context.Background(), testInitData("42"), 12)
	if err != nil || !event.Editable || event.Version != 3 {
		t.Fatal("ordinary owned event not editable", event, err)
	}
	updated, err := client.UpdateEvent(context.Background(), testInitData("42"), 12, 3, input)
	if err != nil || updated.Title != "After" || updated.Version != 4 || !updated.AllDay {
		t.Fatal("ordinary update failed", updated, err)
	}
	for scenario := int32(1); scenario <= 5; scenario++ {
		mode.Store(scenario)
		before := patches.Load()
		event, err := client.GetEvent(context.Background(), testInitData("42"), 12)
		if err != nil || event.Editable {
			t.Fatal("foreign/series/shared/external event editable", scenario, event, err)
		}
		if _, err := client.UpdateEvent(context.Background(), testInitData("42"), 12, 3, input); err == nil || patches.Load() != before {
			t.Fatal("readonly event reached PATCH", scenario)
		}
	}
	mode.Store(0)
	before := patches.Load()
	if _, err := client.UpdateEvent(context.Background(), testInitData("42"), 12, 2, input); err == nil || patches.Load() != before {
		t.Fatal("stale input version reached PATCH")
	}
	race.Store(true)
	_, err = client.UpdateEvent(context.Background(), testInitData("42"), 12, 3, input)
	var apiErr *httpapi.Error
	if !errors.As(err, &apiErr) || apiErr.Code != "day_event_changed" {
		t.Fatal("concurrent upstream mutation not preserved", err)
	}
	invalid := input
	invalid.EndsAt = invalid.StartsAt
	if _, err := NewService(nil, nil).Update(context.Background(), 1, "", 12, 3, invalid); err == nil {
		t.Fatal("update bypassed minimum duration")
	}
}
