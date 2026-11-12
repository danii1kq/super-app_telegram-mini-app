package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/danii1kq/srunal/internal/core/notify"
	"github.com/danii1kq/srunal/internal/platform/config"
	"github.com/danii1kq/srunal/internal/platform/postgres"
	"github.com/danii1kq/srunal/internal/platform/secure"
	"github.com/danii1kq/srunal/internal/platform/telegram"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		slog.Error("config", "err", err)
		os.Exit(1)
	}
	if cfg.BotToken == "" {
		slog.Error("config", "err", "SRUNAL_BOT_TOKEN is required for the worker")
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := postgres.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("database", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	// The worker needs two keys and no more: the delivery key opens the message
	// payload, the identity key opens the Telegram chat id. It never gets the
	// content key, so it cannot read message text or note content.
	deliveryCipher, err := secure.NewCipher(cfg.DeliveryKey)
	if err != nil {
		slog.Error("delivery cipher", "err", err)
		os.Exit(1)
	}
	identityCipher, err := secure.NewCipher(cfg.IdentityKey)
	if err != nil {
		slog.Error("identity cipher", "err", err)
		os.Exit(1)
	}

	worker := notify.NewWorker(pool, telegram.NewClient(cfg.BotToken), deliveryCipher, identityCipher)
	slog.Info("worker started")
	worker.Run(ctx, 2*time.Second)
	slog.Info("worker stopped")
}
