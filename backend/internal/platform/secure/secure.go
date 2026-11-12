// Package secure implements the field-level protection the security model
// requires: AES-256-GCM for private content, SHA-256 for high-entropy tokens
// and HMAC-SHA-256 for the Telegram ID lookup index.
//
// Every sealed value carries its key id and is bound to a context string
// (resource type + id + owner) through the additional authenticated data, so a
// ciphertext copied into another row does not decrypt.
package secure

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

var (
	ErrUnknownKeyID = errors.New("secure: unknown key id")
	ErrDecrypt      = errors.New("secure: decryption failed")
)

// Cipher seals and opens values with one active key and keeps older keys for
// reading during rotation.
type Cipher struct {
	active   keyEntry
	byID     map[string]cipher.AEAD
	activeID string
}

type keyEntry struct {
	id  string
	aead cipher.AEAD
}

// NewCipher builds a cipher from base64-encoded 32-byte keys. The first key is
// the active one; the rest are read-only rotation history.
func NewCipher(keys ...string) (*Cipher, error) {
	if len(keys) == 0 {
		return nil, fmt.Errorf("secure: at least one key is required")
	}
	c := &Cipher{byID: make(map[string]cipher.AEAD, len(keys))}
	for index, encoded := range keys {
		raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(encoded))
		if err != nil {
			return nil, fmt.Errorf("secure: key %d is not valid base64: %w", index, err)
		}
		if len(raw) != 32 {
			return nil, fmt.Errorf("secure: key %d must be 32 bytes, got %d", index, len(raw))
		}
		block, err := aes.NewCipher(raw)
		if err != nil {
			return nil, fmt.Errorf("secure: key %d: %w", index, err)
		}
		aead, err := cipher.NewGCM(block)
		if err != nil {
			return nil, fmt.Errorf("secure: key %d: %w", index, err)
		}
		id := KeyID(raw)
		c.byID[id] = aead
		if index == 0 {
			c.active = keyEntry{id: id, aead: aead}
			c.activeID = id
		}
	}
	return c, nil
}

// KeyID is the stable, non-secret identifier stored next to a ciphertext.
func KeyID(raw []byte) string {
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:8])
}

// Sealed is a ciphertext together with the key that produced it.
type Sealed struct {
	KeyID      string
	Ciphertext []byte
}

// Seal encrypts plaintext bound to aad. The nonce is prepended by the AEAD.
func (c *Cipher) Seal(plaintext []byte, aad string) (Sealed, error) {
	nonce := make([]byte, c.active.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return Sealed{}, fmt.Errorf("secure: nonce: %w", err)
	}
	out := c.active.aead.Seal(nonce, nonce, plaintext, []byte(aad))
	return Sealed{KeyID: c.activeID, Ciphertext: out}, nil
}

// Open decrypts a sealed value, requiring the same aad it was sealed with.
func (c *Cipher) Open(sealed Sealed, aad string) ([]byte, error) {
	aead, ok := c.byID[sealed.KeyID]
	if !ok {
		return nil, ErrUnknownKeyID
	}
	nonceSize := aead.NonceSize()
	if len(sealed.Ciphertext) < nonceSize {
		return nil, ErrDecrypt
	}
	plaintext, err := aead.Open(nil, sealed.Ciphertext[:nonceSize], sealed.Ciphertext[nonceSize:], []byte(aad))
	if err != nil {
		return nil, ErrDecrypt
	}
	return plaintext, nil
}

// SealString is Seal for text fields.
func (c *Cipher) SealString(value, aad string) (Sealed, error) {
	return c.Seal([]byte(value), aad)
}

// OpenString is Open for text fields.
func (c *Cipher) OpenString(sealed Sealed, aad string) (string, error) {
	raw, err := c.Open(sealed, aad)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

// TokenHash is the SHA-256 of a high-entropy random token, hex encoded. Used for
// session tokens and invite secrets: the server never stores the token itself.
func TokenHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// NewToken returns a URL-safe random token with n bytes of entropy.
func NewToken(n int) (string, error) {
	if n < 16 {
		return "", fmt.Errorf("secure: token needs at least 16 bytes of entropy")
	}
	raw := make([]byte, n)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("secure: token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

// LookupIndex is the HMAC-SHA-256 of a value under a dedicated index key. Used
// for the Telegram ID: an ordinary SHA of a numeric id is brute-forceable.
type LookupIndex struct {
	key []byte
}

func NewLookupIndex(base64Key string) (*LookupIndex, error) {
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(base64Key))
	if err != nil {
		return nil, fmt.Errorf("secure: index key is not valid base64: %w", err)
	}
	if len(raw) < 32 {
		return nil, fmt.Errorf("secure: index key must be at least 32 bytes")
	}
	return &LookupIndex{key: raw}, nil
}

func (l *LookupIndex) Of(value string) string {
	mac := hmac.New(sha256.New, l.key)
	mac.Write([]byte(value))
	return hex.EncodeToString(mac.Sum(nil))
}
