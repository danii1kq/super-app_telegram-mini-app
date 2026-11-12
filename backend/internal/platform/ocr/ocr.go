// Package ocr reads a receipt image and returns its lines.
//
// The provider lives behind this interface so the app never depends on one
// vendor, and so a missing key degrades into a clear, translatable error instead
// of a crash. Recognition is a draft: it never decides debts, and the caller
// must let a person check every value.
package ocr

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ErrUnavailable means no provider is configured. The UI offers manual entry.
var ErrUnavailable = errors.New("ocr: provider is not configured")

// ErrUnreadable means the provider answered but could not read the image.
var ErrUnreadable = errors.New("ocr: image could not be read")

// Item is one recognized line. Amount is the price of the whole line in minor
// units; Quantity is how many units the line covers.
type Item struct {
	Name     string
	Quantity int
	Amount   int64
	// Unclear marks a value the recognizer was not sure about. The UI highlights
	// it instead of trusting it.
	Unclear bool
}

// Result is what a receipt looked like to the recognizer.
type Result struct {
	Items []Item
	// Total, Discount and Service are what the receipt printed, in minor units.
	// Zero means the value was not read.
	Total    int64
	Discount int64
	Service  int64
	Currency string
}

// Reader recognizes a receipt image.
type Reader interface {
	Read(ctx context.Context, image []byte, mime string) (Result, error)
}

// Disabled is the reader used when no provider is configured.
type Disabled struct{}

func (Disabled) Read(context.Context, []byte, string) (Result, error) {
	return Result{}, ErrUnavailable
}

// OpenAIRecognizer calls an OpenAI-compatible vision endpoint.
//
// The key stays on the server: it is read from the environment and never
// returned, logged or sent to the client.
type OpenAIRecognizer struct {
	BaseURL string
	APIKey  string
	Model   string
	Client  *http.Client
}

func NewOpenAI(baseURL, apiKey, model string) *OpenAIRecognizer {
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	if model == "" {
		model = "gpt-4o-mini"
	}
	return &OpenAIRecognizer{
		BaseURL: strings.TrimRight(baseURL, "/"),
		APIKey:  apiKey,
		Model:   model,
		Client:  &http.Client{Timeout: 45 * time.Second},
	}
}

// prompt asks for a strict JSON shape. Amounts are requested in minor units so
// no float ever crosses the boundary.
const prompt = `You read a photo of a shop receipt. Reply with JSON only, no prose:
{"currency":"KZT","total_minor":0,"discount_minor":0,"service_minor":0,
 "items":[{"name":"...","quantity":1,"amount_minor":0,"unclear":false}]}
Rules:
- amount_minor is the price of the whole line in minor units (tenge times 100).
- quantity is how many units the line covers.
- Set unclear to true when a value is blurry or unreadable. Never guess: leave
  amount_minor 0 and unclear true instead.
- total_minor, discount_minor and service_minor are what the receipt prints.
  Use 0 when not present or unreadable.
- Do not invent lines that are not on the receipt.`

type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
	// Temperature 0: the same photo should give the same reading.
	Temperature float64 `json:"temperature"`
}

type chatMessage struct {
	Role    string        `json:"role"`
	Content []chatContent `json:"content"`
}

type chatContent struct {
	Type     string    `json:"type"`
	Text     string    `json:"text,omitempty"`
	ImageURL *imageURL `json:"image_url,omitempty"`
}

type imageURL struct {
	URL string `json:"url"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (r *OpenAIRecognizer) Read(ctx context.Context, image []byte, mime string) (Result, error) {
	if r.APIKey == "" {
		return Result{}, ErrUnavailable
	}

	dataURL := "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(image)
	body, err := json.Marshal(chatRequest{
		Model:       r.Model,
		Temperature: 0,
		Messages: []chatMessage{{
			Role: "user",
			Content: []chatContent{
				{Type: "text", Text: prompt},
				{Type: "image_url", ImageURL: &imageURL{URL: dataURL}},
			},
		}},
	})
	if err != nil {
		return Result{}, err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, r.BaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return Result{}, err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+r.APIKey)

	response, err := r.Client.Do(request)
	if err != nil {
		return Result{}, fmt.Errorf("%w: %v", ErrUnreadable, err)
	}
	defer response.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return Result{}, err
	}
	if response.StatusCode != http.StatusOK {
		// The provider's message can echo the request; keep it out of the error
		// the client sees and out of the log.
		return Result{}, fmt.Errorf("%w: status %d", ErrUnreadable, response.StatusCode)
	}

	var parsed chatResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return Result{}, fmt.Errorf("%w: malformed provider response", ErrUnreadable)
	}
	if len(parsed.Choices) == 0 {
		return Result{}, ErrUnreadable
	}

	return parseResult(parsed.Choices[0].Message.Content)
}

// parseResult reads the model's JSON, tolerating a fenced code block.
func parseResult(content string) (Result, error) {
	text := strings.TrimSpace(content)
	if start := strings.Index(text, "{"); start >= 0 {
		if end := strings.LastIndex(text, "}"); end > start {
			text = text[start : end+1]
		}
	}

	var payload struct {
		Currency      string `json:"currency"`
		TotalMinor    int64  `json:"total_minor"`
		DiscountMinor int64  `json:"discount_minor"`
		ServiceMinor  int64  `json:"service_minor"`
		Items         []struct {
			Name        string `json:"name"`
			Quantity    int    `json:"quantity"`
			AmountMinor int64  `json:"amount_minor"`
			Unclear     bool   `json:"unclear"`
		} `json:"items"`
	}
	if err := json.Unmarshal([]byte(text), &payload); err != nil {
		return Result{}, fmt.Errorf("%w: unreadable model output", ErrUnreadable)
	}
	if len(payload.Items) == 0 {
		return Result{}, ErrUnreadable
	}

	result := Result{
		Total:    payload.TotalMinor,
		Discount: payload.DiscountMinor,
		Service:  payload.ServiceMinor,
		Currency: payload.Currency,
		Items:    make([]Item, 0, len(payload.Items)),
	}
	for _, item := range payload.Items {
		quantity := item.Quantity
		if quantity <= 0 {
			quantity = 1
		}
		name := strings.TrimSpace(item.Name)
		unclear := item.Unclear || name == "" || item.AmountMinor <= 0
		if name == "" {
			name = ""
		}
		result.Items = append(result.Items, Item{
			Name:     name,
			Quantity: quantity,
			Amount:   item.AmountMinor,
			Unclear:  unclear,
		})
	}
	return result, nil
}
