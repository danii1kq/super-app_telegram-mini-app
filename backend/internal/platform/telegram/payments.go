package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type InvoicePrice struct {
	Label  string `json:"label"`
	Amount int    `json:"amount"`
}
type InvoiceRequest struct {
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Payload     string         `json:"payload"`
	Currency    string         `json:"currency"`
	Prices      []InvoicePrice `json:"prices"`
}

func (c *Client) paymentCall(ctx context.Context, method string, body any, result any) error {
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/bot"+c.token+"/"+method, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("telegram: invalid payment request")
	}
	req.Header.Set("Content-Type", "application/json")
	response, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("telegram: payment service unavailable")
	}
	defer response.Body.Close()
	data, err = io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return err
	}
	var parsed struct {
		OK     bool            `json:"ok"`
		Result json.RawMessage `json:"result"`
	}
	if err = json.Unmarshal(data, &parsed); err != nil {
		return fmt.Errorf("telegram: invalid payment response")
	}
	if !parsed.OK || response.StatusCode != 200 {
		return fmt.Errorf("telegram: payment request rejected")
	}
	if result != nil {
		return json.Unmarshal(parsed.Result, result)
	}
	return nil
}

func (c *Client) CreateInvoiceLink(ctx context.Context, invoice InvoiceRequest) (string, error) {
	var link string
	err := c.paymentCall(ctx, "createInvoiceLink", invoice, &link)
	return link, err
}
func (c *Client) AnswerPreCheckout(ctx context.Context, id string, ok bool, message string) error {
	return c.paymentCall(ctx, "answerPreCheckoutQuery", map[string]any{"pre_checkout_query_id": id, "ok": ok, "error_message": message}, nil)
}
