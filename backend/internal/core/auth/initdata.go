// Package auth verifies the Telegram Mini App payload and issues sessions.
//
// The rules follow the Telegram documentation: the signature is checked against
// the bot token, auth_date must be fresh, and nothing from initDataUnsafe is
// trusted. The session token is random, lives in memory on the client and is
// stored here only as a SHA-256 hash.
package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

var (
	ErrInvalidSignature = errors.New("auth: invalid initData signature")
	ErrStaleAuth        = errors.New("auth: auth_date is too old")
	ErrMissingUser      = errors.New("auth: initData has no user")
	ErrMissingBotID     = errors.New("auth: initData belongs to another bot")
)

// InitData is the verified subset of the Mini App payload.
type InitData struct {
	TelegramID int64
	Username   string
	FirstName  string
	LastName   string
	PhotoURL   string
	Language   string
	AuthDate   time.Time
	// Raw is the original payload, kept only to present it on the trusted hop
	// to the existing services.
	Raw string
}

// Verifier checks initData against the bot token.
type Verifier struct {
	botToken string
	botID    int64
	maxAge   time.Duration
	now      func() time.Time
}

func NewVerifier(botToken string, botID int64, maxAge time.Duration) *Verifier {
	return &Verifier{botToken: botToken, botID: botID, maxAge: maxAge, now: time.Now}
}

// Verify parses and validates the raw initData string.
func (v *Verifier) Verify(raw string) (InitData, error) {
	values, err := url.ParseQuery(raw)
	if err != nil {
		return InitData{}, ErrInvalidSignature
	}

	hash := values.Get("hash")
	if hash == "" {
		return InitData{}, ErrInvalidSignature
	}

	// The check string is every field except hash, sorted by key, joined with
	// newlines: "key=value" pairs.
	keys := make([]string, 0, len(values))
	for key := range values {
		if key == "hash" || key == "signature" {
			continue
		}
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
		builder.WriteString(values.Get(key))
	}

	secret := hmacSHA256([]byte("WebAppData"), []byte(v.botToken))
	expected := hmacSHA256(secret, []byte(builder.String()))
	if !hmac.Equal([]byte(hex.EncodeToString(expected)), []byte(hash)) {
		return InitData{}, ErrInvalidSignature
	}

	authDateRaw := values.Get("auth_date")
	authUnix, err := strconv.ParseInt(authDateRaw, 10, 64)
	if err != nil {
		return InitData{}, ErrInvalidSignature
	}
	authDate := time.Unix(authUnix, 0).UTC()
	if v.maxAge > 0 && v.now().Sub(authDate) > v.maxAge {
		return InitData{}, ErrStaleAuth
	}

	userRaw := values.Get("user")
	if userRaw == "" {
		return InitData{}, ErrMissingUser
	}
	user, err := parseUser(userRaw)
	if err != nil {
		return InitData{}, ErrMissingUser
	}

	// A payload signed by another bot must not create an account here.
	if v.botID != 0 {
		receiverRaw := values.Get("receiver")
		if receiverRaw != "" {
			receiver, err := parseUser(receiverRaw)
			if err != nil || receiver.ID != v.botID {
				return InitData{}, ErrMissingBotID
			}
		}
	}

	return InitData{
		TelegramID: user.ID,
		Username:   user.Username,
		FirstName:  user.FirstName,
		LastName:   user.LastName,
		PhotoURL:   user.PhotoURL,
		Language:   user.LanguageCode,
		AuthDate:   authDate,
		Raw:        raw,
	}, nil
}

type telegramUser struct {
	ID           int64  `json:"id"`
	Username     string `json:"username"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name"`
	PhotoURL     string `json:"photo_url"`
	LanguageCode string `json:"language_code"`
}

func parseUser(raw string) (telegramUser, error) {
	var user telegramUser
	if err := jsonUnmarshal([]byte(raw), &user); err != nil {
		return telegramUser{}, err
	}
	if user.ID == 0 {
		return telegramUser{}, fmt.Errorf("auth: user id is zero")
	}
	return user, nil
}

func hmacSHA256(key, data []byte) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write(data)
	return mac.Sum(nil)
}
