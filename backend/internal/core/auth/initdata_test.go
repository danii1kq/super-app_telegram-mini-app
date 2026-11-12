package auth

import (
	"encoding/hex"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/danii1kq/srunal/internal/platform/httpapi"
)

const testBotToken = "123456:TEST-TOKEN-for-unit-tests-only"

// sign builds a valid initData string the same way Telegram does.
func sign(t *testing.T, botToken string, fields map[string]string) string {
	t.Helper()
	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var builder strings.Builder
	for index, key := range keys {
		if index > 0 {
			builder.WriteByte('\n')
		}
		builder.WriteString(key)
		builder.WriteByte('=')
		builder.WriteString(fields[key])
	}

	secret := hmacSHA256([]byte("WebAppData"), []byte(botToken))
	hash := hmacSHA256(secret, []byte(builder.String()))

	values := url.Values{}
	for key, value := range fields {
		values.Set(key, value)
	}
	values.Set("hash", hex.EncodeToString(hash))
	return values.Encode()
}

func freshFields() map[string]string {
	return map[string]string{
		"auth_date": strconv.FormatInt(time.Now().Unix(), 10),
		"query_id":  "AAF-test",
		"user":      `{"id":777001,"first_name":"Даниил","username":"danii1kq","language_code":"ru"}`,
	}
}

func TestVerifyAcceptsValidPayload(t *testing.T) {
	verifier := NewVerifier(testBotToken, 0, 24*time.Hour)
	data, err := verifier.Verify(sign(t, testBotToken, freshFields()))
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if data.TelegramID != 777001 {
		t.Fatalf("telegram id = %d", data.TelegramID)
	}
	if data.Username != "danii1kq" {
		t.Fatalf("username = %q", data.Username)
	}
}

func TestVerifyRejectsWrongToken(t *testing.T) {
	verifier := NewVerifier(testBotToken, 0, 24*time.Hour)
	raw := sign(t, "999:OTHER-BOT", freshFields())
	if _, err := verifier.Verify(raw); err != ErrInvalidSignature {
		t.Fatalf("expected ErrInvalidSignature, got %v", err)
	}
}

func TestVerifyRejectsTamperedUser(t *testing.T) {
	verifier := NewVerifier(testBotToken, 0, 24*time.Hour)
	raw := sign(t, testBotToken, freshFields())

	values, err := url.ParseQuery(raw)
	if err != nil {
		t.Fatalf("ParseQuery: %v", err)
	}
	values.Set("user", `{"id":1,"first_name":"Взлом"}`)
	if _, err := verifier.Verify(values.Encode()); err != ErrInvalidSignature {
		t.Fatalf("expected ErrInvalidSignature for a tampered user, got %v", err)
	}
}

func TestVerifyRejectsStaleAuthDate(t *testing.T) {
	verifier := NewVerifier(testBotToken, 0, time.Hour)
	fields := freshFields()
	fields["auth_date"] = strconv.FormatInt(time.Now().Add(-3*time.Hour).Unix(), 10)
	if _, err := verifier.Verify(sign(t, testBotToken, fields)); err != ErrStaleAuth {
		t.Fatalf("expected ErrStaleAuth, got %v", err)
	}
}

func TestVerifyRejectsMissingHash(t *testing.T) {
	verifier := NewVerifier(testBotToken, 0, time.Hour)
	values := url.Values{}
	values.Set("auth_date", strconv.FormatInt(time.Now().Unix(), 10))
	if _, err := verifier.Verify(values.Encode()); err != ErrInvalidSignature {
		t.Fatalf("expected ErrInvalidSignature, got %v", err)
	}
}

func TestBearerToken(t *testing.T) {
	request, _ := http.NewRequest(http.MethodGet, "/api/v1/me", nil)
	if got := BearerToken(request); got != "" {
		t.Fatalf("empty header should give empty token, got %q", got)
	}
	request.Header.Set("Authorization", "Bearer abc123")
	if got := BearerToken(request); got != "abc123" {
		t.Fatalf("BearerToken = %q", got)
	}
}

// The error type is what the client sees; keep the mapping explicit.
func TestUnauthorizedStatus(t *testing.T) {
	if httpapi.Unauthorized("x").Status != http.StatusUnauthorized {
		t.Fatal("Unauthorized must map to 401")
	}
}
