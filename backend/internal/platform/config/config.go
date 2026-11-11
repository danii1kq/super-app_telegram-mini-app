// Package config reads the process configuration from the environment.
//
// Secrets never live in the repository: production supplies them through the
// environment or mounted secret files, development uses a local .env that is
// git-ignored.
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	// HTTP
	Addr string

	// PostgreSQL
	DatabaseURL string

	// Telegram
	BotToken          string
	WebhookSecret     string
	AllowedBotID      int64
	InitDataMaxAge    time.Duration
	SessionTTL        time.Duration
	MiniAppShortName  string

	// Cryptography: base64-encoded 32-byte keys. Empty in development means
	// the process refuses to start unless DevInsecureKeys is set.
	ContentKey     string
	IdentityKey    string
	DeliveryKey    string
	HMACIndexKey   string
	DevInsecureKeys bool
}

func Load() (Config, error) {
	cfg := Config{
		Addr:             env("SRUNAL_ADDR", ":8080"),
		DatabaseURL:      env("SRUNAL_DATABASE_URL", ""),
		BotToken:         env("SRUNAL_BOT_TOKEN", ""),
		WebhookSecret:    env("SRUNAL_WEBHOOK_SECRET", ""),
		AllowedBotID:     envInt("SRUNAL_BOT_ID", 0),
		InitDataMaxAge:   envDuration("SRUNAL_INITDATA_MAX_AGE", 24*time.Hour),
		SessionTTL:       envDuration("SRUNAL_SESSION_TTL", 30*24*time.Hour),
		MiniAppShortName: env("SRUNAL_MINIAPP_NAME", ""),
		ContentKey:       env("SRUNAL_CONTENT_KEY", ""),
		IdentityKey:      env("SRUNAL_IDENTITY_KEY", ""),
		DeliveryKey:      env("SRUNAL_DELIVERY_KEY", ""),
		HMACIndexKey:     env("SRUNAL_HMAC_INDEX_KEY", ""),
		DevInsecureKeys:  envBool("SRUNAL_DEV_INSECURE_KEYS", false),
	}

	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("SRUNAL_DATABASE_URL is required")
	}
	if cfg.ContentKey == "" && !cfg.DevInsecureKeys {
		return Config{}, fmt.Errorf("SRUNAL_CONTENT_KEY is required (or set SRUNAL_DEV_INSECURE_KEYS=1 for local development)")
	}
	return cfg, nil
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envInt(key string, fallback int64) int64 {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return fallback
	}
	return value
}

func envBool(key string, fallback bool) bool {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return fallback
	}
	return value
}

func envDuration(key string, fallback time.Duration) time.Duration {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	value, err := time.ParseDuration(raw)
	if err != nil {
		return fallback
	}
	return value
}
