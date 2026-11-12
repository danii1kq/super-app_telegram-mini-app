package vision

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestVisionRequestAndPrivateFailures(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path != "/chat/completions" || r.Header.Get("Authorization") != "Bearer test-only-secret" {
			t.Error("wrong provider request")
		}
		var body struct {
			Model    string `json:"model"`
			Thinking struct {
				Type string `json:"type"`
			} `json:"thinking"`
			ResponseFormat struct {
				Type string `json:"type"`
			} `json:"response_format"`
			Messages []json.RawMessage `json:"messages"`
		}
		if json.NewDecoder(r.Body).Decode(&body) != nil || body.Model != "deepseek-flash" || body.Thinking.Type != "disabled" || body.ResponseFormat.Type != "json_object" || len(body.Messages) != 2 {
			t.Error("wrong model or thinking enabled")
		}
		if calls == 2 {
			w.WriteHeader(400)
			w.Write([]byte("test-only-secret private image text"))
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]string{"content": "{\"items\":[]}"}, "finish_reason": "stop"}}})
	}))
	defer server.Close()
	client := NewClient(server.URL, "test-only-secret", "")
	result, err := client.Extract(context.Background(), []byte("fixture"), "image/png", "Return JSON items")
	if err != nil || string(result) != "{\"items\":[]}" {
		t.Fatalf("read: %s %v", result, err)
	}
	_, err = client.Extract(context.Background(), []byte("fixture"), "image/png", "Return JSON items")
	if !errors.Is(err, ErrUnreadable) || strings.Contains(err.Error(), "secret") || strings.Contains(err.Error(), "private") {
		t.Fatal("provider error leaked private data")
	}
}

func TestVisionLimitsAndRedirects(t *testing.T) {
	if _, err := NewClient("", "", "").Extract(context.Background(), []byte("x"), "image/png", ""); !errors.Is(err, ErrUnavailable) {
		t.Fatal("missing key accepted")
	}
	for _, target := range []string{"http://example.com", "https://user:pass@example.com", "https://example.com?token=x"} {
		if NewClient(target, "x", "").Configured() {
			t.Fatal("unsafe endpoint accepted")
		}
	}
	reached := false
	destination := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { reached = true }))
	defer destination.Close()
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, destination.URL, 307) }))
	defer source.Close()
	c := NewClient(source.URL, "test-only-secret", "")
	if _, err := c.Extract(context.Background(), []byte("x"), "image/png", ""); !errors.Is(err, ErrUnreadable) || reached {
		t.Fatal("redirect followed")
	}
	for i := 0; i < cap(c.slots); i++ {
		c.slots <- struct{}{}
	}
	if _, err := c.Extract(context.Background(), []byte("x"), "image/png", ""); !errors.Is(err, ErrBusy) {
		t.Fatal("unbounded parallel inference")
	}
	if _, err := c.Extract(context.Background(), []byte("x"), "image/svg+xml", ""); !errors.Is(err, ErrUnreadable) {
		t.Fatal("unsafe image accepted")
	}
}

func TestVisionRejectsBrokenAndTruncatedJSON(t *testing.T) {
	for _, sample := range []struct{ content, finish string }{{"", "stop"}, {"[]", "stop"}, {"{broken}", "stop"}, {"{}", "length"}} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]string{"content": sample.content}, "finish_reason": sample.finish}}})
		}))
		_, err := NewClient(server.URL, "x", "").Extract(context.Background(), []byte("x"), "image/png", "")
		server.Close()
		if !errors.Is(err, ErrUnreadable) {
			t.Fatal("invalid model output accepted")
		}
	}
}
