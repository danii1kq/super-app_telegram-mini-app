package tinder

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/jackc/pgx/v5"

	"github.com/danii1kq/srunal/internal/platform/secure"
)

// notice is the minimal payload a delivery carries.
//
// It names the person and the event only. No bio, no photo, no university:
// a Telegram message sits on a lock screen, and the profile is private until
// the two people match.
type notice struct {
	Text       string `json:"text"`
	ButtonURL  string `json:"button_url"`
	ButtonText string `json:"button_text"`
}

// noticeAAD binds a payload to its recipient and event, so a row cannot be
// replayed against another person.
func tinderNoticeAAD(userID int64, dedupeKey string) string {
	return "tinder_notice:" + strconv.FormatInt(userID, 10) + ":" + dedupeKey
}

// enqueueNotice writes one outbox row inside the caller's transaction.
//
// The row commits with the like or the match, so a rolled-back action never
// leaves a message behind. The dedupe key makes a repeated action a no-op
// instead of a second message.
func (s *Service) enqueueNotice(ctx context.Context, tx pgx.Tx, userID int64, kind, dedupeKey string, body notice) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}
	sealed, err := s.delivery.Seal(payload, tinderNoticeAAD(userID, dedupeKey))
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx,
		`INSERT INTO notification_delivery (user_id, kind, payload_sealed, payload_key_id, dedupe_key)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (user_id, dedupe_key) DO NOTHING`,
		userID, kind, sealed.Ciphertext, sealed.KeyID, dedupeKey,
	)
	return err
}

// notifyLike tells someone that a person liked them.
//
// This is the event the whole app exists for: a like is invisible until the
// other side acts, so without a push the deck is a place where nothing ever
// happens while you are away.
func (s *Service) notifyLike(ctx context.Context, tx pgx.Tx, fromID, toID int64) error {
	enabled, err := s.notificationsEnabled(ctx, tx, toID, "tinder_like_notifications")
	if err != nil || !enabled {
		return err
	}
	name, err := s.displayName(ctx, tx, fromID)
	if err != nil {
		return err
	}
	return s.enqueueNotice(ctx, tx, toID, "tinder.liked",
		"like:"+strconv.FormatInt(fromID, 10)+":"+strconv.FormatInt(toID, 10),
		notice{
			Text:       "SruTinder: " + name + " лайкнул(а) вашу анкету. Откройте приложение, чтобы посмотреть.",
			ButtonURL:  "/apps/tinder",
			ButtonText: "Открыть SruTinder",
		})
}

// notifyMatch tells both sides that the like was mutual.
//
// A match is the one event worth interrupting someone for: it is mutual by
// definition, so neither side is being told about a stranger's private action.
func (s *Service) notifyMatch(ctx context.Context, tx pgx.Tx, aID, bID int64) error {
	for _, pair := range [][2]int64{{aID, bID}, {bID, aID}} {
		recipient, partner := pair[0], pair[1]
		enabled, err := s.notificationsEnabled(ctx, tx, recipient, "tinder_match_notifications")
		if err != nil {
			return err
		}
		if !enabled {
			continue
		}
		name, err := s.displayName(ctx, tx, partner)
		if err != nil {
			return err
		}
		if err := s.enqueueNotice(ctx, tx, recipient, "tinder.matched",
			"match:"+strconv.FormatInt(aID, 10)+":"+strconv.FormatInt(bID, 10)+":"+strconv.FormatInt(recipient, 10),
			notice{
				Text:       "SruTinder: это взаимно! Вы совпали с " + name + ". Напишите первым.",
				ButtonURL:  "/apps/tinder",
				ButtonText: "Открыть SruTinder",
			}); err != nil {
			return err
		}
	}
	return nil
}

// notificationsEnabled reads one switch. A missing settings row means defaults,
// which are on, so a person who never opened settings still gets the message.
func (s *Service) notificationsEnabled(ctx context.Context, tx pgx.Tx, userID int64, column string) (bool, error) {
	// The column name is a constant from this file, never user input.
	query := `SELECT ` + column + ` FROM user_settings WHERE user_id = $1`
	var enabled bool
	err := tx.QueryRow(ctx, query, userID).Scan(&enabled)
	if err != nil {
		if err == pgx.ErrNoRows {
			return true, nil
		}
		return false, err
	}
	return enabled, nil
}

// displayName returns the public name of a person, for the message body.
func (s *Service) displayName(ctx context.Context, tx pgx.Tx, userID int64) (string, error) {
	var name string
	err := tx.QueryRow(ctx, `SELECT public_name FROM users WHERE id = $1`, userID).Scan(&name)
	if err != nil {
		return "", err
	}
	return name, nil
}

// Delivery is the cipher the outbox payloads are sealed with.
//
// It is separate from content on purpose: the worker opens these with the
// delivery key, never with the content key.
func (s *Service) SetDelivery(cipher *secure.Cipher) {
	s.delivery = cipher
}
