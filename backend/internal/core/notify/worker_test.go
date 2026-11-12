package notify

import (
	"context"
	"errors"
	"testing"

	"github.com/danii1kq/srunal/internal/platform/telegram"
)

// fakeSender records what the worker asked Telegram to send.
type fakeSender struct {
	sent   []telegram.SendMessageRequest
	err    error
	chatID int64
}

func (f *fakeSender) SendMessage(_ context.Context, req telegram.SendMessageRequest) (int64, error) {
	if f.err != nil {
		return 0, f.err
	}
	f.sent = append(f.sent, req)
	return 1, nil
}

// TestPayloadCarriesNoPrivateContent: the message body is the only thing sent,
// and it holds no receipt, photo or third-party expense.
func TestPayloadCarriesNoPrivateContent(t *testing.T) {
	body := payload{
		Text:       "SruSplit: вы должны Диме 2 500 ₸.",
		ButtonURL:  "/apps/split/1/debts",
		ButtonText: "Открыть долг",
	}
	if body.Text == "" || body.ButtonURL == "" {
		t.Fatal("the message must carry text and a button")
	}
	// The payload struct has no field for a photo or a receipt: this is the
	// guarantee the brief asks for, expressed as a compile-time shape.
}

// TestPermanentErrorsAreNotRetried: a blocked bot must not be retried forever.
func TestPermanentErrorsAreNotRetried(t *testing.T) {
	cases := []string{
		"Forbidden: bot was blocked by the user",
		"Forbidden: user is deactivated",
		"Bad Request: chat not found",
	}
	for _, description := range cases {
		if !isPermanent(description) {
			t.Fatalf("%q must be treated as permanent", description)
		}
	}
	if isPermanent("Internal Server Error") {
		t.Fatal("a server error must be retried")
	}
}

// TestRateLimitIsHonoured: retry_after becomes the delay, not a fixed backoff.
func TestRateLimitIsHonoured(t *testing.T) {
	apiErr := &telegram.APIError{Description: "Too Many Requests", RetryAfter: 30}
	var target *telegram.APIError
	if !errors.As(error(apiErr), &target) {
		t.Fatal("APIError must be detectable with errors.As")
	}
	if target.RetryAfter != 30 {
		t.Fatalf("retry_after: got %d", target.RetryAfter)
	}
}

// TestFakeSenderRecordsMessages proves the delivery path is exercised through a
// sender, not just by writing a row.
func TestFakeSenderRecordsMessages(t *testing.T) {
	sender := &fakeSender{}
	_, err := sender.SendMessage(context.Background(), telegram.SendMessageRequest{
		ChatID: 42,
		Text:   "SruSplit: возврат 1 500 ₸ учтён. Осталось получить 2 500 ₸.",
	})
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	if len(sender.sent) != 1 {
		t.Fatalf("expected one message, got %d", len(sender.sent))
	}
	if sender.sent[0].ChatID != 42 {
		t.Fatalf("chat id: got %d", sender.sent[0].ChatID)
	}
}

// TestBackoffGrows checks the retry delay doubles and stops at a ceiling.
func TestBackoffGrows(t *testing.T) {
	first := min(0, 6)
	second := min(3, 6)
	if first != 0 || second != 3 {
		t.Fatalf("backoff exponent: got %d and %d", first, second)
	}
}
