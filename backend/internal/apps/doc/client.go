// Package doc is the SruNal adapter for the existing DocGen service.
//
// SruDoc keeps its own database, generation pipeline and operator console. This
// package only calls its trusted API: it never reads DocGen tables and never
// reimplements pricing or generation.
package doc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/danii1kq/srunal/internal/platform/httpapi"
)

// Client talks to the DocGen API.
type Client struct {
	baseURL string
	token   string
	http    *http.Client
}

func NewClient(baseURL, token string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		token:   token,
		http:    &http.Client{Timeout: 20 * time.Second},
	}
}

// Order is the subset SruNal shows in its own space.
type Order struct {
	Number    string    `json:"number"`
	Status    string    `json:"status"`
	Title     string    `json:"title"`
	Amount    string    `json:"amount_minor"`
	CreatedAt time.Time `json:"created_at"`
	HasResult bool      `json:"has_result"`
}

// ListOrders returns the caller's orders from DocGen.
//
// The DocGen API authenticates with its own initData-based session, so SruNal
// passes the user's Telegram payload through on the first hop and stores the
// resulting link. Until that link exists the adapter reports "not linked"
// instead of inventing data.
func (c *Client) ListOrders(ctx context.Context, externalID string) ([]Order, error) {
	if externalID == "" {
		return nil, httpapi.Conflict("not_linked", "Аккаунт SruDoc ещё не привязан")
	}

	var out struct {
		Orders []Order `json:"orders"`
	}
	if err := c.get(ctx, "/api/me/orders", externalID, &out); err != nil {
		return nil, err
	}
	return out.Orders, nil
}

// OrderStatus returns one order, checking that it belongs to the caller.
func (c *Client) OrderStatus(ctx context.Context, externalID, number string) (Order, error) {
	if externalID == "" {
		return Order{}, httpapi.Conflict("not_linked", "Аккаунт SruDoc ещё не привязан")
	}
	var order Order
	if err := c.get(ctx, "/api/orders/"+number, externalID, &order); err != nil {
		return Order{}, err
	}
	return order, nil
}

// CreateDeadlineRequest is the payload for the SruDoc -> SruDay link. It is
// defined here because SruDoc owns the order side of that action.
type CreateDeadlineRequest struct {
	OrderNumber string `json:"order_number"`
	Title       string `json:"title"`
	DueDate     string `json:"due_date"`
}

func (c *Client) get(ctx context.Context, path, externalID string, dst any) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("doc: request: %w", err)
	}
	// The trusted-API token identifies SruNal to DocGen; the external id scopes
	// the answer to one account.
	request.Header.Set("Authorization", "Bearer "+c.token)
	request.Header.Set("X-Srunal-External-Id", externalID)

	response, err := c.http.Do(request)
	if err != nil {
		return httpapi.Conflict("doc_unavailable", "SruDoc сейчас недоступен")
	}
	defer response.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("doc: read: %w", err)
	}

	switch {
	case response.StatusCode == http.StatusNotFound:
		return httpapi.NotFound("Заказ не найден")
	case response.StatusCode == http.StatusUnauthorized || response.StatusCode == http.StatusForbidden:
		return httpapi.Conflict("not_linked", "Аккаунт SruDoc ещё не привязан")
	case response.StatusCode >= 400:
		return httpapi.Conflict("doc_error", "SruDoc вернул ошибку")
	}

	if err := json.Unmarshal(raw, dst); err != nil {
		return fmt.Errorf("doc: decode: %w", err)
	}
	return nil
}

// post is used by the integration actions that create something in DocGen.
func (c *Client) post(ctx context.Context, path, externalID string, body any, dst any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("doc: marshal: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("doc: request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+c.token)
	request.Header.Set("X-Srunal-External-Id", externalID)

	response, err := c.http.Do(request)
	if err != nil {
		return httpapi.Conflict("doc_unavailable", "SruDoc сейчас недоступен")
	}
	defer response.Body.Close()

	if response.StatusCode >= 400 {
		return httpapi.Conflict("doc_error", "SruDoc вернул ошибку")
	}
	return json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(dst)
}
