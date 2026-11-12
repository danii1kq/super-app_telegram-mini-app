// Package config reads the process configuration from the environment.
//
// Secrets never live in the repository: production supplies them through the
// environment or mounted secret files, development uses a local .env that is
// git-ignored.
package config

import (
	"crypto/sha256"
	"encoding/base64"
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
	BotToken         string
	FarmStarsEnabled bool
	WebhookSecret    string
	AllowedBotID     int64
	InitDataMaxAge   time.Duration
	SessionTTL       time.Duration
	MiniAppShortName string

	// Existing services SruNal adapts to.
	DocAPIURL   string
	DocAPIToken string
	DayAPIURL   string
	DayAPIToken string

	// Receipt recognition. Empty key means the feature reports "unavailable"
	// and the UI offers manual entry; it never falls back to fake results.
	OCRBaseURL string
	OCRAPIKey  string
	OCRModel   string

	// Cryptography: base64-encoded 32-byte keys. Empty in development means
	// the process refuses to start unless DevInsecureKeys is set.
	ContentKey      string
	IdentityKey     string
	DeliveryKey     string
	HMACIndexKey    string
	DevInsecureKeys bool
}

func Load() (Config, error) {
	cfg := Config{
		Addr:             env("SRUNAL_ADDR", ":8080"),
		DatabaseURL:      env("SRUNAL_DATABASE_URL", ""),
		BotToken:         env("SRUNAL_BOT_TOKEN", ""),
		FarmStarsEnabled: envBool("SRUNAL_FARM_STARS_ENABLED", false),
		WebhookSecret:    env("SRUNAL_WEBHOOK_SECRET", ""),
		AllowedBotID:     envInt("SRUNAL_BOT_ID", 0),
		InitDataMaxAge:   envDuration("SRUNAL_INITDATA_MAX_AGE", 24*time.Hour),
		SessionTTL:       envDuration("SRUNAL_SESSION_TTL", 30*24*time.Hour),
		MiniAppShortName: env("SRUNAL_MINIAPP_NAME", ""),
		DocAPIURL:        env("SRUNAL_DOC_API_URL", ""),
		DocAPIToken:      env("SRUNAL_DOC_API_TOKEN", ""),
		DayAPIURL:        env("SRUNAL_DAY_API_URL", ""),
		DayAPIToken:      env("SRUNAL_DAY_API_TOKEN", ""),
		OCRBaseURL:       env("SRUNAL_OCR_BASE_URL", ""),
		OCRAPIKey:        env("SRUNAL_OCR_API_KEY", ""),
		OCRModel:         env("SRUNAL_OCR_MODEL", ""),
		ContentKey:       env("SRUNAL_CONTENT_KEY", ""),
		IdentityKey:      env("SRUNAL_IDENTITY_KEY", ""),
		DeliveryKey:      env("SRUNAL_DELIVERY_KEY", ""),
		HMACIndexKey:     env("SRUNAL_HMAC_INDEX_KEY", ""),
		DevInsecureKeys:  envBool("SRUNAL_DEV_INSECURE_KEYS", false),
	}

	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("SRUNAL_DATABASE_URL is required")
	}
	if cfg.FarmStarsEnabled && (cfg.BotToken == "" || cfg.WebhookSecret == "") {
		return Config{}, fmt.Errorf("farm Stars require SRUNAL_BOT_TOKEN and SRUNAL_WEBHOOK_SECRET")
	}

	if cfg.DevInsecureKeys {
		// Local development without a secret store: generate ephemeral keys so the
		// process starts. They are lost on restart, which is the point: nothing
		// encrypted in a dev database survives a restart, and no key ever lands
		// in the repository.
		for name, target := range map[string]*string{
			"SRUNAL_CONTENT_KEY":    &cfg.ContentKey,
			"SRUNAL_IDENTITY_KEY":   &cfg.IdentityKey,
			"SRUNAL_DELIVERY_KEY":   &cfg.DeliveryKey,
			"SRUNAL_HMAC_INDEX_KEY": &cfg.HMACIndexKey,
		} {
			if *target == "" {
				key, err := devKey()
				if err != nil {
					return Config{}, fmt.Errorf("generate %s: %w", name, err)
				}
				*target = key
			}
		}
	}

	if cfg.ContentKey == "" {
		return Config{}, fmt.Errorf("SRUNAL_CONTENT_KEY is required (or set SRUNAL_DEV_INSECURE_KEYS=1 for local development)")
	}
	if cfg.HMACIndexKey == "" {
		return Config{}, fmt.Errorf("SRUNAL_HMAC_INDEX_KEY is required (or set SRUNAL_DEV_INSECURE_KEYS=1 for local development)")
	}
	return cfg, nil
}

// devKey returns a base64-encoded 32-byte key for local development only.
//
// Deterministic on purpose: a random key per process would make everything
// encrypted in the dev database unreadable after a restart. This value is public
// and must never be used outside SRUNAL_DEV_INSECURE_KEYS=1.
func devKey() (string, error) {
	sum := sha256.Sum256([]byte("srunal-dev-insecure-key"))
	return base64.StdEncoding.EncodeToString(sum[:]), nil
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
