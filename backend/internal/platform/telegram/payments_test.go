package telegram

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStarsInvoiceAndCheckout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if r.URL.Path == "/bottest-token/createInvoiceLink" {
			if body["currency"] != "XTR" || body["provider_token"] != nil {
				t.Error("Stars invoice must omit provider token")
			}
			if len(body["prices"].([]any)) != 1 {
				t.Error("Stars invoice needs one price")
			}
			_, _ = w.Write([]byte(`{"ok":true,"result":"https://t.me/$fixture"}`))
		} else if r.URL.Path == "/bottest-token/answerPreCheckoutQuery" {
			if body["pre_checkout_query_id"] != "q1" || body["ok"] != true {
				t.Error("wrong checkout answer")
			}
			_, _ = w.Write([]byte(`{"ok":true,"result":true}`))
		} else {
			t.Error("unexpected method")
			w.WriteHeader(404)
		}
	}))
	defer server.Close()
	bot := NewClient("test-token")
	bot.SetBaseURL(server.URL)
	link, err := bot.CreateInvoiceLink(context.Background(), InvoiceRequest{Title: "Theme", Description: "Theme", Payload: "fixture", Currency: "XTR", Prices: []InvoicePrice{{Label: "Theme", Amount: 150}}})
	if err != nil || link != "https://t.me/$fixture" {
		t.Fatalf("invalid invoice result: %s %v", link, err)
	}
	if err = bot.AnswerPreCheckout(context.Background(), "q1", true, ""); err != nil {
		t.Fatal(err)
	}
}
