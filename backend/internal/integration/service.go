// Package integration owns the confirmed actions between applications.
//
// It is the only place that combines two apps: it calls the public services of
// the owner of each object, never their stores. Every action is idempotent by a
// client key, and the confirmation is bound to the content the user saw.
package integration

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strconv"

	"github.com/jackc/pgx/v5"

	"github.com/danii1kq/srunal/internal/apps/day"
	"github.com/danii1kq/srunal/internal/apps/doc"
	"github.com/danii1kq/srunal/internal/apps/notes"
	"github.com/danii1kq/srunal/internal/core/auth"
	"github.com/danii1kq/srunal/internal/platform/httpapi"
	"github.com/danii1kq/srunal/internal/platform/postgres"
)

type Service struct {
	pool  *postgres.Pool
	doc   *doc.Service
	day   *day.Service
	notes *notes.Service
	auth  *auth.Service
}

func NewService(pool *postgres.Pool, docService *doc.Service, dayService *day.Service, notesService *notes.Service, authService *auth.Service) *Service {
	return &Service{pool: pool, doc: docService, day: dayService, notes: notesService, auth: authService}
}

// DeadlineFromOrder creates a SruDay event for a SruDoc order.
//
// The caller must own the order: the doc adapter answers only for the linked
// account. The content hash makes a repeated request with the same key return
// the stored result, and the same key with different content a conflict.
func (s *Service) DeadlineFromOrder(ctx context.Context, userID int64, initData, idempotencyKey, orderNumber, title, dueDate string) (map[string]any, error) {
	if idempotencyKey == "" {
		return nil, httpapi.BadRequest("missing_idempotency_key", "Не передан ключ идемпотентности")
	}
	if orderNumber == "" || dueDate == "" {
		return nil, httpapi.Validation(map[string]string{"order_number": "Выберите заказ и дату"}, "Проверьте поля")
	}

	// The order must belong to the caller; the adapter enforces that.
	order, err := s.doc.Order(ctx, userID, orderNumber)
	if err != nil {
		return nil, err
	}

	content := map[string]string{
		"order": orderNumber,
		"title": title,
		"due":   dueDate,
	}
	contentHash, err := hashContent(content)
	if err != nil {
		return nil, httpapi.Internal(err)
	}

	// A stored result for the same key returns as-is; a different hash is a
	// conflict, because the user confirmed different content.
	var storedResult []byte
	var storedHash string
	err = s.pool.QueryRow(ctx,
		`SELECT content_hash, result FROM integration_requests
		  WHERE idempotency_key = $1 AND user_id = $2`, idempotencyKey, userID,
	).Scan(&storedHash, &storedResult)
	if err == nil {
		if storedHash != contentHash {
			return nil, httpapi.Conflict("idempotency_conflict", "Этот ключ уже использован с другими данными")
		}
		var out map[string]any
		if err := json.Unmarshal(storedResult, &out); err != nil {
			return nil, httpapi.Internal(err)
		}
		return out, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	eventTitle := title
	if eventTitle == "" {
		eventTitle = "Дедлайн: " + order.Title
	}

	event, err := s.day.CreateDeadline(ctx, userID, initData, day.CreateEventInput{
		Title:     eventTitle,
		StartsAt:  dueDate,
		EndsAt:    dueDate,
		AllDay:    true,
		SourceRef: "doc:" + orderNumber,
	})
	if err != nil {
		// The source object stays untouched: only the calendar action failed.
		return nil, err
	}

	result := map[string]any{
		"event_id":     event.ID,
		"event_title":  event.Title,
		"order_number": orderNumber,
		"due_date":     dueDate,
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return nil, httpapi.Internal(err)
	}

	if _, err := s.pool.Exec(ctx,
		`INSERT INTO integration_requests (idempotency_key, user_id, source_app, target_app, content_hash, result)
		 VALUES ($1, $2, 'doc', 'day', $3, $4)
		 ON CONFLICT (idempotency_key) DO NOTHING`,
		idempotencyKey, userID, contentHash, encoded,
	); err != nil {
		return nil, err
	}

	return result, nil
}

// ShareNoteCard grants read access and returns the card the chat stores.
//
// The card holds the note id, not the text: every open re-checks the grant, so a
// revoked note stops being readable even through an old message.
func (s *Service) ShareNoteCard(ctx context.Context, userID, noteID int64, recipients []int64) (map[string]any, error) {
	if len(recipients) == 0 {
		return nil, httpapi.Validation(map[string]string{"recipients": "Выберите получателей"}, "Проверьте поля")
	}

	// Only the author can share; the notes service enforces it.
	if err := s.notes.Grant(ctx, userID, noteID, recipients, "direct", nil); err != nil {
		return nil, err
	}

	note, err := s.notes.Get(ctx, userID, noteID)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"kind":  "note_card",
		"app":   "notes",
		"id":    strconv.FormatInt(noteID, 10),
		"title": note.Title,
	}, nil
}

func hashContent(value any) (string, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:]), nil
}
