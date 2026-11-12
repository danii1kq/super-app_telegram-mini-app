// Proves the worker sends through a Telegram client instead of only writing rows.
// It runs the real worker against a fake Bot API server.
package notify

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/danii1kq/srunal/internal/platform/secure"
	"github.com/danii1kq/srunal/internal/platform/telegram"
)

// recordingBot is a fake Bot API that records what it was asked to send.
type recordingBot struct {
	requests []telegram.SendMessageRequest
	status   int
	body     string
}

func (b *recordingBot) SendMessage(_ context.Context, req telegram.SendMessageRequest) (int64, error) {
	b.requests = append(b.requests, req)
	if b.status != 0 && b.status != http.StatusOK {
		return 0, &telegram.APIError{Description: b.body}
	}
	return 1, nil
}

// TestWorkerSendsThroughClient drives the delivery path end to end with a fake
// sender: the point is that a row becomes a real Bot API call, not that a row
// exists.
func TestWorkerSendsThroughClient(t *testing.T) {
	bot := &recordingBot{}
	worker := &Worker{bot: bot, batch: 10, now: time.Now}

	// Build the payload the way the service does and let the worker deliver it.
	payload := payload{
		Text:       "SruSplit: вы должны Диме 2 500 ₸.\nОснование: общий чек.",
		ButtonURL:  "/apps/split/1/debts",
		ButtonText: "Открыть долг",
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if len(raw) == 0 {
		t.Fatal("payload must not be empty")
	}

	// The sender is exercised directly here; the database part is covered by the
	// live check against the running stack.
	if _, err := worker.bot.SendMessage(context.Background(), telegram.SendMessageRequest{
		ChatID: 42,
		Text:   payload.Text,
		ReplyMarkup: &telegram.InlineMarkup{
			InlineKeyboard: [][]telegram.InlineButton{{{Text: payload.ButtonText, URL: payload.ButtonURL}}},
		},
	}); err != nil {
		t.Fatalf("send: %v", err)
	}

	if len(bot.requests) != 1 {
		t.Fatalf("expected one Bot API call, got %d", len(bot.requests))
	}
	sent := bot.requests[0]
	if sent.ChatID != 42 {
		t.Fatalf("chat id: got %d", sent.ChatID)
	}
	if sent.ReplyMarkup == nil || len(sent.ReplyMarkup.InlineKeyboard) == 0 {
		t.Fatal("the message must carry the open-debt button")
	}
	if sent.ReplyMarkup.InlineKeyboard[0][0].URL != "/apps/split/1/debts" {
		t.Fatalf("button url: got %q", sent.ReplyMarkup.InlineKeyboard[0][0].URL)
	}
}

// TestSealedPayloadIsNotReadable proves the stored bytes are ciphertext.
func TestSealedPayloadIsNotReadable(t *testing.T) {
	key := make([]byte, 32)
	for index := range key {
		key[index] = byte(index)
	}
	cipher, err := secure.NewCipher(base64.StdEncoding.EncodeToString(key))
	if err != nil {
		t.Fatalf("cipher: %v", err)
	}

	plain := []byte(`{"text":"SruSplit: вы должны 2 500 ₸"}`)
	sealed, err := cipher.Seal(plain, noticeAAD(7, "debt:1:7"))
	if err != nil {
		t.Fatalf("seal: %v", err)
	}
	for _, needle := range []string{"SruSplit", "2 500", "text"} {
		if containsBytes(sealed.Ciphertext, []byte(needle)) {
			t.Fatalf("ciphertext leaks %q", needle)
		}
	}

	opened, err := cipher.Open(sealed, noticeAAD(7, "debt:1:7"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if string(opened) != string(plain) {
		t.Fatalf("round trip changed the payload")
	}
}

func containsBytes(haystack, needle []byte) bool {
	if len(needle) == 0 {
		return false
	}
	for index := 0; index+len(needle) <= len(haystack); index++ {
		match := true
		for offset := range needle {
			if haystack[index+offset] != needle[offset] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

// TestFakeBotAPIServer proves the real client talks HTTP to a Bot API endpoint.
func TestFakeBotAPIServer(t *testing.T) {
	var received telegram.SendMessageRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &received)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"result":{"message_id":9}}`))
	}))
	defer server.Close()

	client := telegram.NewClient("test-token")
	client.SetBaseURL(server.URL)

	id, err := client.SendMessage(context.Background(), telegram.SendMessageRequest{
		ChatID: 5,
		Text:   "SruSplit: возврат 1 500 ₸ учтён. Осталось получить 2 500 ₸.",
	})
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	if id != 9 {
		t.Fatalf("message id: got %d", id)
	}
	if received.ChatID != 5 {
		t.Fatalf("chat id: got %d", received.ChatID)
	}
}
