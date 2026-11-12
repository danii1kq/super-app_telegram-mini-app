package farm

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/danii1kq/srunal/internal/core/auth"
	"github.com/danii1kq/srunal/internal/platform/httpapi"
	"github.com/danii1kq/srunal/internal/platform/secure"
	"github.com/jackc/pgx/v5"
)

type Coinflip struct {
	ID         string    `json:"id"`
	HostID     string    `json:"host_id"`
	HostName   string    `json:"host_name"`
	GuestID    string    `json:"guest_id"`
	GuestName  string    `json:"guest_name"`
	Stake      int       `json:"stake"`
	HostSide   string    `json:"host_side"`
	ResultSide string    `json:"result_side"`
	WinnerID   string    `json:"winner_id"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
	ExpiresAt  time.Time `json:"expires_at"`
}

func lockedGamePlayer(ctx context.Context, tx pgx.Tx, userID int64) (State, error) {
	var data []byte
	err := tx.QueryRow(ctx, `SELECT state FROM farm_players WHERE user_id=$1 FOR UPDATE`, userID).Scan(&data)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return State{}, httpapi.BadRequest("coinflip", "Сначала открой ферму")
		}
		return State{}, err
	}
	var state State
	if err = json.Unmarshal(data, &state); err != nil {
		return state, err
	}
	normalizeState(&state)
	refreshCrash(&state, time.Now().UTC())
	return state, nil
}

func saveGamePlayer(ctx context.Context, tx pgx.Tx, userID int64, state State) error {
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `UPDATE farm_players SET state=$2,updated_at=now() WHERE user_id=$1`, userID, data)
	return err
}

// Offer rows are always locked before account rows; paired accounts use ID order.
func (h *Handler) expireCoinflips(ctx context.Context) error {
	rows, err := h.pool.Query(ctx, `SELECT id FROM farm_coinflips WHERE status='open' AND expires_at<=now() ORDER BY expires_at LIMIT 50`)
	if err != nil {
		return err
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		ids = append(ids, id)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	for _, id := range ids {
		err = h.pool.InTx(ctx, func(tx pgx.Tx) error {
			var host int64
			var stake int
			if err := tx.QueryRow(ctx, `SELECT host_id,stake FROM farm_coinflips WHERE id=$1 AND status='open' AND expires_at<=now() FOR UPDATE SKIP LOCKED`, id).Scan(&host, &stake); err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return nil
				}
				return err
			}
			state, err := lockedGamePlayer(ctx, tx, host)
			if err != nil {
				return err
			}
			state.Coins += stake
			if err = saveGamePlayer(ctx, tx, host, state); err != nil {
				return err
			}
			_, err = tx.Exec(ctx, `UPDATE farm_coinflips SET status='expired',resolved_at=now() WHERE id=$1`, id)
			return err
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (h *Handler) coinflipList(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFrom(r.Context())
	if err := h.expireCoinflips(r.Context()); err != nil {
		httpapi.WriteError(w, err)
		return
	}
	rows, err := h.pool.Query(r.Context(), `SELECT f.id,f.host_id::text,h.public_name,coalesce(f.guest_id::text,''),coalesce(g.public_name,''),f.stake,f.host_side,f.result_side,coalesce(f.winner_id::text,''),f.status,f.created_at,f.expires_at
 FROM farm_coinflips f JOIN users h ON h.id=f.host_id LEFT JOIN users g ON g.id=f.guest_id
 WHERE f.status='open' OR (f.host_id=$1 OR f.guest_id=$1) AND f.created_at>now()-interval '7 days'
 ORDER BY (f.host_id=$1 AND f.status='open') DESC,(f.status='open') DESC,f.created_at DESC LIMIT 40`, user.ID)
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}
	defer rows.Close()
	flips := []Coinflip{}
	for rows.Next() {
		var flip Coinflip
		if err = rows.Scan(&flip.ID, &flip.HostID, &flip.HostName, &flip.GuestID, &flip.GuestName, &flip.Stake, &flip.HostSide, &flip.ResultSide, &flip.WinnerID, &flip.Status, &flip.CreatedAt, &flip.ExpiresAt); err != nil {
			httpapi.WriteError(w, err)
			return
		}
		flips = append(flips, flip)
	}
	if err = rows.Err(); err != nil {
		httpapi.WriteError(w, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, map[string]any{"flips": flips, "server_time": time.Now().UTC()})
}

func (h *Handler) coinflipAction(w http.ResponseWriter, r *http.Request, action Action) {
	user, _ := auth.UserFrom(r.Context())
	ctx := r.Context()
	if err := h.expireCoinflips(ctx); err != nil {
		httpapi.WriteError(w, err)
		return
	}
	var mine State
	message := ""
	err := h.pool.InTx(ctx, func(tx pgx.Tx) error {
		bad := func(message string) error { return httpapi.BadRequest("coinflip", message) }
		if action.Kind == "flip_create" {
			if !validStake(action.Stake) {
				return bad("Ставка должна быть от 100 до 100 000 монет")
			}
			if action.Side != "heads" && action.Side != "tails" {
				return bad("Выбери солнце или луну")
			}
			var err error
			mine, err = lockedGamePlayer(ctx, tx, user.ID)
			if err != nil {
				return err
			}
			if err = checkGameRequest(mine, action.RequestID); err != nil {
				return err
			}
			var open bool
			if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM farm_coinflips WHERE host_id=$1 AND status='open')`, user.ID).Scan(&open); err != nil {
				return err
			}
			if open {
				return bad("У тебя уже есть открытый коинфлип")
			}
			if mine.Coins < action.Stake {
				return bad("Недостаточно монет")
			}
			id, err := secure.NewToken(16)
			if err != nil {
				return err
			}
			now := time.Now().UTC()
			if _, err = tx.Exec(ctx, `INSERT INTO farm_coinflips(id,host_id,stake,host_side,status,created_at,expires_at) VALUES($1,$2,$3,$4,'open',$5,$6)`, id, user.ID, action.Stake, action.Side, now, now.Add(10*time.Minute)); err != nil {
				return err
			}
			mine.Coins -= action.Stake
			rememberGameRequest(&mine, action.RequestID)
			message = "Ставка внесена. Ждём настоящего соперника"
			return saveGamePlayer(ctx, tx, user.ID, mine)
		}
		var host int64
		var stake int
		var side, status string
		var expires time.Time
		if err := tx.QueryRow(ctx, `SELECT host_id,stake,host_side,status,expires_at FROM farm_coinflips WHERE id=$1 FOR UPDATE`, action.Item).Scan(&host, &stake, &side, &status, &expires); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return bad("Этот коинфлип не найден")
			}
			return err
		}
		if status != "open" || !expires.After(time.Now().UTC()) {
			return bad("Коинфлип уже закрыт. Обнови список")
		}
		if action.Kind == "flip_cancel" {
			if host != user.ID {
				return bad("Отменить можно только свою заявку")
			}
			var err error
			mine, err = lockedGamePlayer(ctx, tx, user.ID)
			if err != nil {
				return err
			}
			if err = checkGameRequest(mine, action.RequestID); err != nil {
				return err
			}
			mine.Coins += stake
			rememberGameRequest(&mine, action.RequestID)
			if err = saveGamePlayer(ctx, tx, user.ID, mine); err != nil {
				return err
			}
			_, err = tx.Exec(ctx, `UPDATE farm_coinflips SET status='cancelled',resolved_at=now() WHERE id=$1`, action.Item)
			message = "Коинфлип отменён, ставка возвращена"
			return err
		}
		if action.Kind != "flip_join" {
			return bad("Неизвестное действие")
		}
		if host == user.ID {
			return bad("Нельзя играть против себя")
		}
		states := map[int64]State{}
		for _, id := range []int64{min(host, user.ID), max(host, user.ID)} {
			state, err := lockedGamePlayer(ctx, tx, id)
			if err != nil {
				return err
			}
			states[id] = state
		}
		guestState := states[user.ID]
		if err := checkGameRequest(guestState, action.RequestID); err != nil {
			return err
		}
		if guestState.Coins < stake {
			return bad("Недостаточно монет для этой ставки")
		}
		value, err := random(2)
		if err != nil {
			return err
		}
		resultSide := "heads"
		if value == 1 {
			resultSide = "tails"
		}
		winner := host
		if resultSide != side {
			winner = user.ID
		}
		payout := stake * 180 / 100
		now := time.Now().UTC()
		for id, state := range states {
			if id == user.ID {
				state.Coins -= stake
				rememberGameRequest(&state, action.RequestID)
			}
			result := GameResult{ID: action.Item, Kind: "flip", Status: "lost", Stake: stake, At: now, Side: resultSide}
			if id == winner {
				state.Coins += payout
				result.Payout = payout
				result.Status = "won"
			}
			appendGame(&state, result)
			if err = saveGamePlayer(ctx, tx, id, state); err != nil {
				return err
			}
			if id == user.ID {
				mine = state
			}
		}
		_, err = tx.Exec(ctx, `UPDATE farm_coinflips SET guest_id=$2,result_side=$3,winner_id=$4,status='done',resolved_at=now() WHERE id=$1`, action.Item, user.ID, resultSide, winner)
		if winner == user.ID {
			message = "Коинфлип выигран! Банк зачислен"
		} else {
			message = "Коинфлип завершён. В этот раз победил соперник"
		}
		return err
	})
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}
	h.writePlayer(w, mine, time.Now().UTC(), message, nil)
}

func (h *Handler) upgradeQuote(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFrom(r.Context())
	var action Action
	if err := httpapi.DecodeJSON(w, r, &action, 2048); err != nil {
		httpapi.WriteError(w, err)
		return
	}
	var data []byte
	if err := h.pool.QueryRow(r.Context(), `SELECT state FROM farm_players WHERE user_id=$1`, user.ID).Scan(&data); err != nil {
		httpapi.WriteError(w, err)
		return
	}
	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		httpapi.WriteError(w, err)
		return
	}
	normalizeState(&state)
	quote, err := plantUpgradeQuote(state, action)
	if err != nil {
		httpapi.WriteError(w, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, quote)
}
