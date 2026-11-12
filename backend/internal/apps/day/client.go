// Package day adapts the existing SruDay API without reading its database.
package day

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/danii1kq/srunal/internal/platform/httpapi"
)

type Client struct {
	baseURL  string
	http     *http.Client
	mu       sync.Mutex
	sessions map[[32]byte]daySession
}

type daySession struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	User      struct {
		ID         int64 `json:"id"`
		TelegramID int64 `json:"telegram_id"`
	} `json:"user"`
}

// A deployment token is not a user's SruDay session and is never forwarded as one.
func NewClient(baseURL, _ string) *Client {
	return &Client{baseURL: strings.TrimRight(baseURL, "/"), sessions: make(map[[32]byte]daySession),
		http: &http.Client{Timeout: 20 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}
}

func (c *Client) Configured() bool {
	u, err := url.Parse(c.baseURL)
	return err == nil && u.Host != "" && u.User == nil && (u.Scheme == "https" || u.Scheme == "http") && u.RawQuery == "" && u.Fragment == ""
}

type Event struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	StartsAt  time.Time `json:"starts_at"`
	EndsAt    time.Time `json:"ends_at"`
	AllDay    bool      `json:"all_day"`
	Location  string    `json:"location,omitempty"`
	SourceRef string    `json:"source_ref,omitempty"`
}

type CreateEventInput struct {
	Title     string `json:"title"`
	StartsAt  string `json:"starts_at"`
	EndsAt    string `json:"ends_at"`
	AllDay    bool   `json:"all_day"`
	Location  string `json:"location,omitempty"`
	SourceRef string `json:"source_ref,omitempty"`
}

// authenticate exchanges an already verified payload for the upstream's user session.
func (c *Client) authenticate(ctx context.Context, initData string) (daySession, error) {
	values, err := url.ParseQuery(initData)
	var user struct {
		ID int64 `json:"id"`
	}
	if err != nil || json.Unmarshal([]byte(values.Get("user")), &user) != nil || user.ID <= 0 {
		return daySession{}, httpapi.Conflict("day_reauth_required", "Откройте SruNal заново в Telegram для подключения SruDay")
	}
	key := sha256.Sum256([]byte(initData))
	c.mu.Lock()
	session, found := c.sessions[key]
	c.mu.Unlock()
	if found && session.ExpiresAt.After(time.Now()) {
		return session, nil
	}
	if err := c.do(ctx, http.MethodPost, "/api/v1/auth/telegram", "", map[string]string{"init_data": initData}, &session); err != nil {
		return daySession{}, err
	}
	if session.Token == "" || session.User.ID <= 0 || session.User.TelegramID != user.ID || !session.ExpiresAt.After(time.Now()) {
		return daySession{}, httpapi.Conflict("day_identity_mismatch", "SruDay не подтвердил тот же Telegram-аккаунт")
	}
	// Short, bounded memory caching avoids creating an upstream session on each week change.
	if until := time.Now().Add(10 * time.Minute); session.ExpiresAt.After(until) {
		session.ExpiresAt = until
	}
	c.mu.Lock()
	if len(c.sessions) >= 256 {
		for savedKey, saved := range c.sessions {
			if !saved.ExpiresAt.After(time.Now()) {
				delete(c.sessions, savedKey)
			}
		}
		if len(c.sessions) >= 256 {
			for savedKey := range c.sessions {
				delete(c.sessions, savedKey)
				break
			}
		}
	}
	c.sessions[key] = session
	c.mu.Unlock()
	return session, nil
}

func (c *Client) ListEvents(ctx context.Context, initData string, from, to time.Time) ([]Event, error) {
	session, err := c.authenticate(ctx, initData)
	if err != nil {
		return nil, err
	}
	query := url.Values{"from": {from.UTC().Format(time.RFC3339)}, "to": {to.UTC().Format(time.RFC3339)}}
	var out struct {
		Occurrences []struct {
			EventID  int64     `json:"event_id"`
			Title    string    `json:"title"`
			StartsAt time.Time `json:"starts_at"`
			EndsAt   time.Time `json:"ends_at"`
			AllDay   bool      `json:"all_day"`
			Location string    `json:"location"`
		} `json:"occurrences"`
	}
	if err := c.do(ctx, http.MethodGet, "/api/v1/events?"+query.Encode(), session.Token, nil, &out); err != nil {
		c.forget(initData)
		return nil, err
	}
	if out.Occurrences == nil {
		return nil, httpapi.Conflict("day_error", "SruDay не вернул список событий")
	}
	events := make([]Event, 0, len(out.Occurrences))
	for _, item := range out.Occurrences {
		if item.EventID <= 0 || item.StartsAt.IsZero() || !item.EndsAt.After(item.StartsAt) {
			return nil, httpapi.Conflict("day_error", "SruDay вернул некорректное событие")
		}
		events = append(events, Event{ID: strconv.FormatInt(item.EventID, 10), Title: item.Title, StartsAt: item.StartsAt, EndsAt: item.EndsAt, AllDay: item.AllDay, Location: item.Location})
	}
	return events, nil
}

func (c *Client) CreateEvent(ctx context.Context, initData string, input CreateEventInput) (Event, error) {
	session, err := c.authenticate(ctx, initData)
	if err != nil {
		return Event{}, err
	}
	// The upstream has no source_ref/idempotency support. Never retry this POST automatically.
	var out struct {
		ID       int64     `json:"id"`
		Title    string    `json:"title"`
		StartsAt time.Time `json:"starts_at"`
		EndsAt   time.Time `json:"ends_at"`
		AllDay   bool      `json:"all_day"`
		Location string    `json:"location"`
	}
	body := map[string]any{"title": input.Title, "starts_at": input.StartsAt, "ends_at": input.EndsAt, "all_day": input.AllDay, "location": input.Location, "visibility": "private"}
	if err := c.do(ctx, http.MethodPost, "/api/v1/events", session.Token, body, &out); err != nil {
		c.forget(initData)
		return Event{}, err
	}
	if out.ID <= 0 || out.Title == "" || out.StartsAt.IsZero() || !out.EndsAt.After(out.StartsAt) {
		return Event{}, httpapi.Conflict("day_error", "SruDay не подтвердил создание события. Проверьте календарь перед повтором")
	}
	return Event{ID: strconv.FormatInt(out.ID, 10), Title: out.Title, StartsAt: out.StartsAt, EndsAt: out.EndsAt, AllDay: out.AllDay, Location: out.Location, SourceRef: input.SourceRef}, nil
}

func (c *Client) forget(initData string) {
	c.mu.Lock()
	delete(c.sessions, sha256.Sum256([]byte(initData)))
	c.mu.Unlock()
}

func (c *Client) do(ctx context.Context, method, path, token string, body any, dst any) error {
	if !c.Configured() {
		return httpapi.Conflict("day_unavailable", "Подключение к SruDay ещё не настроено")
	}
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return httpapi.Conflict("day_unavailable", "SruDay сейчас недоступен")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	response, err := c.http.Do(req)
	if err != nil {
		return httpapi.Conflict("day_unavailable", "SruDay сейчас недоступен. Перед повтором создания проверьте календарь")
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusUnauthorized || response.StatusCode == http.StatusForbidden {
		return httpapi.Conflict("day_reauth_required", "SruDay не подтвердил вход. Нужны свежие данные Telegram и совместимый бот")
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return httpapi.Conflict("day_error", "SruDay отклонил запрос. Проверьте календарь перед повтором")
	}
	raw, err := io.ReadAll(io.LimitReader(response.Body, (1<<20)+1))
	if err != nil || len(raw) > 1<<20 {
		return httpapi.Conflict("day_error", "Не удалось прочитать ответ SruDay")
	}
	if json.Unmarshal(raw, dst) != nil {
		return httpapi.Conflict("day_error", "Некорректный ответ SruDay")
	}
	return nil
}
