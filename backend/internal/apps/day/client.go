// Package day is the SruNal adapter for the existing SruDay service.
//
// SruDay owns events, repetitions, calendar groups and availability. SruNal
// calls its /api/v1 routes with the user's verified Telegram payload and never
// reads SruDay tables.
package day

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/danii1kq/srunal/internal/platform/httpapi"
)

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

// Event is the subset SruNal shows.
type Event struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	StartsAt  time.Time `json:"starts_at"`
	EndsAt    time.Time `json:"ends_at"`
	AllDay    bool      `json:"all_day"`
	Location  string    `json:"location,omitempty"`
	SourceRef string    `json:"source_ref,omitempty"`
}

// ListEvents returns events in a window.
func (c *Client) ListEvents(ctx context.Context, initData string, from, to time.Time) ([]Event, error) {
	query := url.Values{}
	query.Set("from", from.UTC().Format(time.RFC3339))
	query.Set("to", to.UTC().Format(time.RFC3339))

	var out struct {
		Events []Event `json:"events"`
	}
	if err := c.do(ctx, http.MethodGet, "/api/v1/events?"+query.Encode(), initData, nil, &out); err != nil {
		return nil, err
	}
	return out.Events, nil
}

// CreateEventInput is the payload for a new event.
type CreateEventInput struct {
	Title     string `json:"title"`
	StartsAt  string `json:"starts_at"`
	EndsAt    string `json:"ends_at"`
	AllDay    bool   `json:"all_day"`
	Location  string `json:"location,omitempty"`
	SourceRef string `json:"source_ref,omitempty"`
}

// CreateEvent creates one event in the caller's calendar.
//
// The source_ref lets SruNal recognise an event it created, so a repeated
// integration request does not duplicate it.
func (c *Client) CreateEvent(ctx context.Context, initData string, input CreateEventInput) (Event, error) {
	var event Event
	if err := c.do(ctx, http.MethodPost, "/api/v1/events", initData, input, &event); err != nil {
		return Event{}, err
	}
	return event, nil
}

// Availability returns the caller's free windows.
func (c *Client) Availability(ctx context.Context, initData string, from, to time.Time) ([]Event, error) {
	query := url.Values{}
	query.Set("from", from.UTC().Format(time.RFC3339))
	query.Set("to", to.UTC().Format(time.RFC3339))

	var out struct {
		Windows []Event `json:"windows"`
	}
	if err := c.do(ctx, http.MethodGet, "/api/v1/events/availability?"+query.Encode(), initData, nil, &out); err != nil {
		return nil, err
	}
	return out.Windows, nil
}

func (c *Client) do(ctx context.Context, method, path, initData string, body any, dst any) error {
	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("day: marshal: %w", err)
		}
		reader = bytes.NewReader(payload)
	}

	request, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return fmt.Errorf("day: request: %w", err)
	}
	// SruDay authenticates with the Telegram payload on the trusted hop.
	request.Header.Set("X-Telegram-Init-Data", initData)
	if c.token != "" {
		request.Header.Set("Authorization", "Bearer "+c.token)
	}
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}

	response, err := c.http.Do(request)
	if err != nil {
		return httpapi.Conflict("day_unavailable", "SruDay сейчас недоступен")
	}
	defer response.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("day: read: %w", err)
	}

	switch {
	case response.StatusCode == http.StatusUnauthorized || response.StatusCode == http.StatusForbidden:
		return httpapi.Conflict("not_linked", "Аккаунт SruDay ещё не привязан")
	case response.StatusCode == http.StatusNotFound:
		return httpapi.NotFound("Событие не найдено")
	case response.StatusCode >= 400:
		return httpapi.Conflict("day_error", "SruDay вернул ошибку")
	}

	if dst == nil {
		return nil
	}
	if err := json.Unmarshal(raw, dst); err != nil {
		return fmt.Errorf("day: decode: %w", err)
	}
	return nil
}
