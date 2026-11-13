package farm

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/danii1kq/srunal/internal/core/auth"
	"github.com/danii1kq/srunal/internal/platform/httpapi"
	"github.com/danii1kq/srunal/internal/platform/secure"
	"github.com/danii1kq/srunal/internal/platform/telegram"
	"github.com/jackc/pgx/v5"
)

type PaymentBot interface {
	CreateInvoiceLink(context.Context, telegram.InvoiceRequest) (string, error)
	AnswerPreCheckout(context.Context, string, bool, string) error
}
type Stars struct {
	bot    PaymentBot
	index  *secure.LookupIndex
	secret string
}

func (h *Handler) EnableStars(bot PaymentBot, index *secure.LookupIndex, secret string) {
	if bot != nil && index != nil && secret != "" {
		h.stars = &Stars{bot, index, secret}
	}
}

func (h *Handler) invoice(w http.ResponseWriter, r *http.Request) {
	if h.stars == nil {
		httpapi.WriteError(w, httpapi.BadRequest("stars_unavailable", "Оплата Stars пока не подключена. Можно использовать промокод."))
		return
	}
	var body struct {
		Theme string `json:"theme"`
	}
	if err := httpapi.DecodeJSON(w, r, &body, 1024); err != nil {
		httpapi.WriteError(w, err)
		return
	}
	theme, ok := findTheme(body.Theme)
	if !ok || theme.Stars == 0 {
		httpapi.WriteError(w, httpapi.BadRequest("theme", "Тема не найдена"))
		return
	}
	user, _ := auth.UserFrom(r.Context())
	var lookup string
	var owned bool
	err := h.pool.QueryRow(r.Context(), `SELECT u.telegram_id_lookup,coalesce(f.state->'owned_themes' ? $2,false) FROM users u LEFT JOIN farm_players f ON f.user_id=u.id WHERE u.id=$1`, user.ID, theme.ID).Scan(&lookup, &owned)
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}
	if owned {
		httpapi.WriteError(w, httpapi.BadRequest("theme_owned", "Тема уже открыта"))
		return
	}
	payload, err := secure.NewToken(24)
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}
	_, err = h.pool.Exec(r.Context(), `INSERT INTO farm_theme_orders(payload_hash,user_id,payer_lookup,theme,amount) VALUES($1,$2,$3,$4,$5)`, secure.TokenHash(payload), user.ID, lookup, theme.ID, theme.Stars)
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}
	link, err := h.stars.bot.CreateInvoiceLink(r.Context(), telegram.InvoiceRequest{Title: theme.Name, Description: "Оформление фермы SruFarm. Постоянный доступ к теме.", Payload: payload, Currency: "XTR", Prices: []telegram.InvoicePrice{{Label: theme.Name, Amount: theme.Stars}}})
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}
	httpapi.WriteJSON(w, 200, map[string]string{"url": link})
}

type checkout struct {
	ID   string `json:"id"`
	From struct {
		ID int64 `json:"id"`
	} `json:"from"`
	Currency string `json:"currency"`
	Amount   int    `json:"total_amount"`
	Payload  string `json:"invoice_payload"`
}
type paidTheme struct {
	Currency string `json:"currency"`
	Amount   int    `json:"total_amount"`
	Payload  string `json:"invoice_payload"`
	Charge   string `json:"telegram_payment_charge_id"`
}
type paymentUpdate struct {
	Checkout *checkout `json:"pre_checkout_query"`
	Message  *struct {
		From struct {
			ID int64 `json:"id"`
		} `json:"from"`
		Payment *paidTheme `json:"successful_payment"`
	} `json:"message"`
}

func (h *Handler) paymentWebhook(w http.ResponseWriter, r *http.Request) {
	if h.stars == nil || subtle.ConstantTimeCompare([]byte(r.Header.Get("X-Telegram-Bot-Api-Secret-Token")), []byte(h.stars.secret)) != 1 {
		httpapi.WriteError(w, httpapi.Unauthorized("Неверный webhook"))
		return
	}
	var update paymentUpdate
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&update); err != nil {
		httpapi.WriteError(w, httpapi.BadRequest("update", "Некорректное обновление"))
		return
	}
	if update.Checkout != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		q := update.Checkout
		var valid bool
		err := h.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM farm_theme_orders WHERE payload_hash=$1 AND payer_lookup=$2 AND amount=$3 AND paid_at IS NULL AND created_at>now()-interval '20 minutes')`, secure.TokenHash(q.Payload), h.stars.index.Of(strconv.FormatInt(q.From.ID, 10)), q.Amount).Scan(&valid)
		if err != nil {
			httpapi.WriteError(w, err)
			return
		}
		valid = valid && q.Currency == "XTR"
		message := ""
		if !valid {
			message = "Этот счёт недействителен. Открой новый счёт в SruFarm."
		}
		if err = h.stars.bot.AnswerPreCheckout(ctx, q.ID, valid, message); err != nil {
			httpapi.WriteError(w, err)
			return
		}
	}
	if update.Message != nil && update.Message.Payment != nil {
		if err := h.completePayment(r.Context(), update.Message.From.ID, *update.Message.Payment); err != nil {
			httpapi.WriteError(w, err)
			return
		}
	}
	httpapi.WriteJSON(w, 200, map[string]bool{"ok": true})
}

func (h *Handler) completePayment(ctx context.Context, payer int64, payment paidTheme) error {
	if payment.Currency != "XTR" || payment.Charge == "" {
		return httpapi.BadRequest("payment", "Некорректная оплата")
	}
	return h.pool.InTx(ctx, func(tx pgx.Tx) error {
		var userID int64
		var theme, lookup string
		var amount int
		var charge *string
		err := tx.QueryRow(ctx, `SELECT user_id,theme,payer_lookup,amount,charge_id FROM farm_theme_orders WHERE payload_hash=$1 FOR UPDATE`, secure.TokenHash(payment.Payload)).Scan(&userID, &theme, &lookup, &amount, &charge)
		if errors.Is(err, pgx.ErrNoRows) {
			return httpapi.NotFound("Счёт не найден")
		}
		if err != nil {
			return err
		}
		if lookup != h.stars.index.Of(strconv.FormatInt(payer, 10)) || amount != payment.Amount {
			return httpapi.BadRequest("payment", "Оплата не соответствует счёту")
		}
		if charge != nil {
			if *charge == payment.Charge {
				return nil
			}
			return httpapi.Conflict("payment", "Счёт уже оплачен")
		}
		initial, err := json.Marshal(initialState(time.Now()))
		if err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `INSERT INTO farm_players(user_id,state) VALUES($1,$2) ON CONFLICT DO NOTHING`, userID, initial); err != nil {
			return err
		}
		var data []byte
		var state State
		if err = tx.QueryRow(ctx, `SELECT state FROM farm_players WHERE user_id=$1 FOR UPDATE`, userID).Scan(&data); err != nil {
			return err
		}
		if err = json.Unmarshal(data, &state); err != nil {
			return err
		}
		normalizeState(&state)
		unlockTheme(&state, theme)
		data, err = json.Marshal(state)
		if err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `UPDATE farm_players SET state=$2,updated_at=now() WHERE user_id=$1`, userID, data); err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `UPDATE farm_theme_orders SET paid_at=now(),charge_id=$2 WHERE payload_hash=$1`, secure.TokenHash(payment.Payload), payment.Charge)
		return err
	})
}
