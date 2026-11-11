package secure

import (
	"encoding/base64"
	"testing"
)

func testKey(t *testing.T, fill byte) string {
	t.Helper()
	raw := make([]byte, 32)
	for i := range raw {
		raw[i] = fill
	}
	return base64.StdEncoding.EncodeToString(raw)
}

func TestSealOpenRoundTrip(t *testing.T) {
	cipher, err := NewCipher(testKey(t, 1))
	if err != nil {
		t.Fatalf("NewCipher: %v", err)
	}

	sealed, err := cipher.SealString("привет, SruNal", "note:42:owner:7")
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	if sealed.KeyID == "" {
		t.Fatal("KeyID must be set")
	}

	got, err := cipher.OpenString(sealed, "note:42:owner:7")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if got != "привет, SruNal" {
		t.Fatalf("round trip mismatch: %q", got)
	}
}

// A ciphertext copied into another row must not decrypt: that is the whole
// point of binding the AAD to the resource.
func TestOpenRejectsForeignContext(t *testing.T) {
	cipher, err := NewCipher(testKey(t, 2))
	if err != nil {
		t.Fatalf("NewCipher: %v", err)
	}
	sealed, err := cipher.SealString("secret", "note:1:owner:1")
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	if _, err := cipher.OpenString(sealed, "note:2:owner:1"); err == nil {
		t.Fatal("expected decryption failure for a foreign context")
	}
}

func TestOpenRejectsUnknownKey(t *testing.T) {
	cipher, err := NewCipher(testKey(t, 3))
	if err != nil {
		t.Fatalf("NewCipher: %v", err)
	}
	sealed, err := cipher.SealString("secret", "aad")
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	sealed.KeyID = "deadbeefdeadbeef"
	if _, err := cipher.OpenString(sealed, "aad"); err != ErrUnknownKeyID {
		t.Fatalf("expected ErrUnknownKeyID, got %v", err)
	}
}

func TestNewCipherRejectsBadKeyLength(t *testing.T) {
	short := base64.StdEncoding.EncodeToString([]byte("too-short"))
	if _, err := NewCipher(short); err == nil {
		t.Fatal("expected an error for a key that is not 32 bytes")
	}
}

func TestTokenHashIsStableAndNotTheToken(t *testing.T) {
	token, err := NewToken(32)
	if err != nil {
		t.Fatalf("NewToken: %v", err)
	}
	if len(token) < 32 {
		t.Fatalf("token too short: %q", token)
	}
	first := TokenHash(token)
	if first != TokenHash(token) {
		t.Fatal("TokenHash must be deterministic")
	}
	if first == token {
		t.Fatal("TokenHash must not return the token")
	}
}

func TestLookupIndexDiffersFromPlainSHA(t *testing.T) {
	index, err := NewLookupIndex(testKey(t, 4))
	if err != nil {
		t.Fatalf("NewLookupIndex: %v", err)
	}
	if index.Of("123456789") == TokenHash("123456789") {
		t.Fatal("HMAC index must not equal a plain SHA-256")
	}
	if index.Of("123456789") != index.Of("123456789") {
		t.Fatal("lookup index must be deterministic")
	}
}
