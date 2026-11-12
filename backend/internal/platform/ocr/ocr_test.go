package ocr

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/danii1kq/srunal/internal/platform/vision"
)

const validReceipt = `{"currency":"KZT","total_minor":600000,"discount_minor":0,"service_minor":0,"items":[{"name":"Пицца","quantity":1,"amount_minor":400000,"unclear":false},{"name":"Бургер","quantity":1,"amount_minor":200000,"unclear":false}]}`

func TestDeepSeekReceiptDraft(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" || r.Header.Get("Authorization") != "Bearer test-key" {
			t.Error("missing private provider request")
		}
		var body struct {
			Model    string `json:"model"`
			Thinking struct {
				Type string `json:"type"`
			} `json:"thinking"`
			Messages []json.RawMessage `json:"messages"`
		}
		if json.NewDecoder(r.Body).Decode(&body) != nil || body.Model != "deepseek-flash" || body.Thinking.Type != "disabled" || len(body.Messages) != 2 {
			t.Error("unexpected model request")
		}
		json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"finish_reason": "stop", "message": map[string]string{"content": validReceipt}}}})
	}))
	defer server.Close()
	result, err := NewDeepSeek(vision.NewClient(server.URL, "test-key", "")).Read(context.Background(), []byte("test-image"), "image/jpeg")
	if err != nil || result.Total != 600000 || len(result.Items) != 2 || result.Items[0].Amount != 400000 {
		t.Fatalf("draft: %#v %v", result, err)
	}
	if _, err := NewDeepSeek(nil).Read(context.Background(), nil, ""); !errors.Is(err, ErrUnavailable) {
		t.Fatal("missing provider must offer manual entry")
	}
}

func TestReceiptOutputValidation(t *testing.T) {
	for name, raw := range map[string]string{
		"foreign currency": strings.Replace(validReceipt, `"KZT"`, `"USD"`, 1),
		"unknown currency": strings.Replace(validReceipt, `"KZT"`, `""`, 1),
		"negative line":    strings.Replace(validReceipt, `400000`, `-1`, 1),
		"decimal money":    strings.Replace(validReceipt, `400000`, `4.5`, 1),
		"oversized money":  strings.Replace(validReceipt, `400000`, `1000000000001`, 1),
		"too many units":   strings.Replace(validReceipt, `"quantity":1`, `"quantity":1000`, 1),
		"zero units":       strings.Replace(validReceipt, `"quantity":1`, `"quantity":0`, 1),
		"wrong schema":     strings.Replace(validReceipt, `"currency"`, `"injected"`, 1),
		"trailing payload": validReceipt + `{}`,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := parseResult([]byte(raw)); !errors.Is(err, ErrUnreadable) {
				t.Fatalf("accepted invalid output: %v", err)
			}
		})
	}
	unknown := strings.Replace(validReceipt, `"quantity":1`, `"quantity":null`, 1)
	result, err := parseResult([]byte(unknown))
	if err != nil || !result.Items[0].Unclear || result.Items[0].Quantity != 1 {
		t.Fatal("unknown quantity must remain visibly unclear")
	}
	lines := strings.Repeat(`{"name":"x","quantity":1,"amount_minor":1,"unclear":false},`, MaxItems) + `{"name":"last","quantity":1,"amount_minor":1,"unclear":false}`
	if _, err := parseResult([]byte(`{"currency":"KZT","items":[` + lines + `]}`)); !errors.Is(err, ErrUnreadable) {
		t.Fatal("accepted too many lines")
	}
	if _, err := parseResult([]byte(strings.Replace(validReceipt, `"Пицца"`, `"`+strings.Repeat("x", 201)+`"`, 1))); err == nil {
		t.Fatal("accepted too long item name")
	}
}
