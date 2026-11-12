package farm

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/danii1kq/srunal/internal/core/auth"
	"github.com/danii1kq/srunal/internal/platform/httpapi"
	"github.com/danii1kq/srunal/internal/platform/postgres"
	"github.com/jackc/pgx/v5"
)

type Handler struct {
	pool  *postgres.Pool
	auth  *auth.Service
	stars *Stars
}

func NewHandler(pool *postgres.Pool, authService *auth.Service) *Handler {
	return &Handler{pool: pool, auth: authService}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.Handle("GET /api/v1/farm", h.auth.Middleware(http.HandlerFunc(h.state)))
	mux.Handle("POST /api/v1/farm/actions", h.auth.Middleware(http.HandlerFunc(h.action)))
	mux.Handle("GET /api/v1/farm/leaderboard", h.auth.Middleware(http.HandlerFunc(h.leaderboard)))
	mux.Handle("GET /api/v1/farm/coinflips", h.auth.Middleware(http.HandlerFunc(h.coinflipList)))
	mux.Handle("POST /api/v1/farm/upgrade-quote", h.auth.Middleware(http.HandlerFunc(h.upgradeQuote)))
	mux.Handle("POST /api/v1/farm/themes/invoice", h.auth.Middleware(http.HandlerFunc(h.invoice)))
	if h.stars != nil {
		mux.HandleFunc("POST /bot/webhook", h.paymentWebhook)
	}
}

func (h *Handler) state(w http.ResponseWriter, r *http.Request) { h.player(w, r, nil) }

func (h *Handler) action(w http.ResponseWriter, r *http.Request) {
	var action Action
	if err := httpapi.DecodeJSON(w, r, &action, 2048); err != nil {
		httpapi.WriteError(w, err)
		return
	}
	if action.Kind == "flip_create" || action.Kind == "flip_join" || action.Kind == "flip_cancel" {
		h.coinflipAction(w, r, action)
		return
	}
	h.player(w, r, &action)
}

func (h *Handler) player(w http.ResponseWriter, r *http.Request, action *Action) {
	user, _ := auth.UserFrom(r.Context())
	now := time.Now().UTC()
	var state State
	message := ""
	drops := []string{}
	err := h.pool.InTx(r.Context(), func(tx pgx.Tx) error {
		initial, err := json.Marshal(initialState(now))
		if err != nil {
			return err
		}
		if _, err = tx.Exec(r.Context(), `INSERT INTO farm_players (user_id,state) VALUES ($1,$2) ON CONFLICT DO NOTHING`, user.ID, initial); err != nil {
			return err
		}
		var data []byte
		if err = tx.QueryRow(r.Context(), `SELECT state FROM farm_players WHERE user_id=$1 FOR UPDATE`, user.ID).Scan(&data); err != nil {
			return err
		}
		if err = json.Unmarshal(data, &state); err != nil {
			return err
		}
		normalizeState(&state)
		now = time.Now().UTC()
		refreshOrders(&state, user.ID, now)
		changed := false
		if action == nil || action.Kind != "crash_cashout" {
			changed = refreshCrash(&state, now)
		}
		if action == nil && !changed {
			return nil
		}
		var revenue int
		if action != nil {
			if action.Kind == "redeem" {
				err = redeemTheme(r.Context(), tx, user.ID, &state, action.Code)
				message = "Тема открыта и применена"
			} else {
				message, drops, revenue, err = apply(&state, *action, now, random)
			}
		}
		if err != nil {
			return err
		}
		data, err = json.Marshal(state)
		if err != nil {
			return err
		}
		if _, err = tx.Exec(r.Context(), `UPDATE farm_players SET state=$2,updated_at=now() WHERE user_id=$1`, user.ID, data); err != nil {
			return err
		}
		if revenue > 0 {
			_, err = tx.Exec(r.Context(), `INSERT INTO farm_weekly_sales (week,user_id,revenue) VALUES ($1,$2,$3)
			ON CONFLICT (week,user_id) DO UPDATE SET revenue=farm_weekly_sales.revenue+EXCLUDED.revenue`, weekStart(now), user.ID, revenue)
		}
		return err
	})
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}
	h.writePlayer(w, state, now, message, drops)
}

func (h *Handler) writePlayer(w http.ResponseWriter, state State, now time.Time, message string, drops []string) {
	level, into, need := levelProgress(state.XP)
	state.Orders = quotedOrders(state)
	state = publicGameState(state, now)
	httpapi.WriteJSON(w, http.StatusOK, map[string]any{"state": state, "crops": Crops, "recipes": Recipes, "mutations": mutationCatalog(state.Upgrades["genetics"]), "packs": packsFor(state.Upgrades["selection"]), "upgrades": upgradesFor(state), "themes": Themes, "stars_available": h.stars != nil, "server_time": now, "message": message, "drops": drops,
		"level": map[string]any{"level": level, "title": titleFor(level), "into": into, "need": need, "max": MaxLevel, "pending": claimableLevels(state), "rewards": levelCatalog(state)},
		"daily": dailyCatalog(state, now), "expeditions": expeditionCatalog(), "games": gameRules()})
}

func weekStart(now time.Time) time.Time {
	now = now.UTC()
	day := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	return day.AddDate(0, 0, -(int(day.Weekday())+6)%7)
}

type ranking struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Revenue int64  `json:"revenue"`
	Rank    int    `json:"rank"`
	Mine    bool   `json:"mine"`
}

func (h *Handler) leaderboard(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFrom(r.Context())
	week := weekStart(time.Now())
	// Include the current player even when they are outside the first 50 places.
	rows, err := h.pool.Query(r.Context(), `WITH ranked AS (
	 SELECT s.user_id,u.public_name,s.revenue,row_number() OVER (ORDER BY s.revenue DESC,s.user_id) AS rank
	 FROM farm_weekly_sales s JOIN users u ON u.id=s.user_id WHERE s.week=$1
	) SELECT user_id,public_name,revenue,rank FROM ranked WHERE rank<=50 OR user_id=$2 ORDER BY rank`, week, user.ID)
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}
	defer rows.Close()
	entries := make([]ranking, 0)
	for rows.Next() {
		var id int64
		var item ranking
		if err := rows.Scan(&id, &item.Name, &item.Revenue, &item.Rank); err != nil {
			httpapi.WriteError(w, err)
			return
		}
		item.ID = strconv.FormatInt(id, 10)
		item.Mine = id == user.ID
		entries = append(entries, item)
	}
	if err := rows.Err(); err != nil {
		httpapi.WriteError(w, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, map[string]any{"entries": entries, "ends_at": week.AddDate(0, 0, 7)})
}
