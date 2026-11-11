// Package notify owns the notification outbox and the Telegram delivery worker.
package notify

import (
	"context"
	"log/slog"
	"time"

	"github.com/danii1kq/srunal/internal/platform/postgres"
	"github.com/danii1kq/srunal/internal/platform/secure"
	"github.com/danii1kq/srunal/internal/platform/telegram"
)

// Worker drains notification_delivery rows and sends them through the Bot API.
type Worker struct {
	pool     *postgres.Pool
	bot      *telegram.Client
	delivery *secure.Cipher
}

func NewWorker(pool *postgres.Pool, bot *telegram.Client, delivery *secure.Cipher) *Worker {
	return &Worker{pool: pool, bot: bot, delivery: delivery}
}

// Run polls the outbox until ctx is cancelled.
func (w *Worker) Run(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.tick(ctx); err != nil {
				slog.Error("notify tick", "err", err)
			}
		}
	}
}

// tick is a no-op until the outbox table exists (feat/core-social).
func (w *Worker) tick(ctx context.Context) error {
	_ = ctx
	return nil
}
