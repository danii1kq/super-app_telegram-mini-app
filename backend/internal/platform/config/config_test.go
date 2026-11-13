package config

import "testing"

func TestWorkerAndMigrationsDoNotRequireContentKeys(t *testing.T) {
	t.Setenv("SRUNAL_DATABASE_URL", "postgres://localhost/test")
	t.Setenv("SRUNAL_DEV_INSECURE_KEYS", "")
	t.Setenv("SRUNAL_CONTENT_KEY", "")
	t.Setenv("SRUNAL_HMAC_INDEX_KEY", "")
	t.Setenv("SRUNAL_IDENTITY_KEY", "identity")
	t.Setenv("SRUNAL_DELIVERY_KEY", "delivery")
	cfg, err := LoadWorker()
	if err != nil || cfg.ContentKey != "" || cfg.HMACIndexKey != "" {
		t.Fatal("worker requires unrelated keys", err)
	}
	if _, err = Load(); err == nil {
		t.Fatal("API accepted missing keys")
	}
	t.Setenv("SRUNAL_IDENTITY_KEY", "")
	t.Setenv("SRUNAL_DELIVERY_KEY", "")
	if _, err = LoadDatabase(); err != nil {
		t.Fatal("migration requires encryption keys", err)
	}
	if _, err = LoadWorker(); err == nil {
		t.Fatal("worker accepted missing identity and delivery keys")
	}
}
