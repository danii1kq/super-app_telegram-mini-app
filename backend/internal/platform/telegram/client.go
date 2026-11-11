// Package telegram wraps the Bot API calls the worker and the webhook need.
package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	token string
	http  *http.Client
}

func NewClient(token string) *Client {
	return &Client{
		token: token,
		http:  &http.Client{Timeout: 15 * time.Second},
	}
}

type SendMessageRequest struct {
	ChatID      int64         `json:"chat_id"`
	Text        string        `json:"text"`
	ParseMode   string        `json:"parse_mode,omitempty"`
	ReplyMarkup *InlineMarkup `json:"reply_markup,omitempty"`
}

type InlineMarkup struct {
	InlineKeyboard [][]InlineButton `json:"inline_keyboard"`
}

type InlineButton struct {
	Text         string `json:"text"`
	URL          string `json:"url,omitempty"`
	CallbackData string `json:"callback_data,omitempty"`
}

// SendMessage posts a message and returns the Telegram message id.
func (c *Client) SendMessage(ctx context.Context, req SendMessageRequest) (int64, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return 0, fmt.Errorf("telegram: marshal: %w", err)
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", c.token)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return 0, fmt.Errorf("telegram: request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return 0, fmt.Errorf("telegram: send: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return 0, fmt.Errorf("telegram: read: %w", err)
	}

	var parsed struct {
		OK          bool   `json:"ok"`
		Description string `json:"description"`
		Result      struct {
			MessageID int64 `json:"message_id"`
		} `json:"result"`
		Parameters struct {
			RetryAfter int `json:"retry_after"`
		} `json:"parameters"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return 0, fmt.Errorf("telegram: decode: %w", err)
	}
	if !parsed.OK {
		// The description is a Telegram-side error string, safe to log; the URL
		// with the token never is.
		return 0, &APIError{Description: parsed.Description, RetryAfter: parsed.Parameters.RetryAfter}
	}
	return parsed.Result.MessageID, nil
}

// APIError is a non-ok Bot API answer. RetryAfter is set when Telegram asked to
// slow down.
type APIError struct {
	Description string
	RetryAfter  int
}

func (e *APIError) Error() string { return "telegram: " + e.Description }
