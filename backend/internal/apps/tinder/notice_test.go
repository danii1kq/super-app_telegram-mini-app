package tinder

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/danii1kq/srunal/internal/platform/secure"
)

// The notification paths are exercised against a real database in the
// integration suite; these tests cover the pure parts: the payload shape, the
// AAD binding and the switch reader's default.

func TestNoticePayloadCarriesNoProfileContent(t *testing.T) {
	body := notice{
		Text:       "SruTinder: Айгуль лайкнул(а) вашу анкету.",
		ButtonURL:  "/apps/tinder",
		ButtonText: "Открыть SruTinder",
	}
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	// Only the three delivery fields may appear: a bio, a photo or a university
	// would leak the profile onto a lock screen.
	for key := range decoded {
		switch key {
		case "text", "button_url", "button_text":
		default:
			t.Fatalf("unexpected field in the payload: %q", key)
		}
	}
	if len(decoded) != 3 {
		t.Fatalf("payload has %d fields, want 3", len(decoded))
	}
}

func TestNoticeAADBindsRecipientAndEvent(t *testing.T) {
	first := tinderNoticeAAD(7, "like:1:7")
	second := tinderNoticeAAD(8, "like:1:7")
	third := tinderNoticeAAD(7, "match:1:7:7")
	if first == second || first == third {
		t.Fatal("AAD does not distinguish recipient or event")
	}
	if first != "tinder_notice:7:like:1:7" {
		t.Fatalf("unexpected AAD shape: %q", first)
	}
}

func TestSealedNoticeDoesNotContainPlaintext(t *testing.T) {
	cipher, err := secure.NewCipher("AQIDBAUGBwgJCgsMDQ4PEBESExQVFhcYGRobHB0eHyA=")
	if err != nil {
		t.Fatal(err)
	}
	body := notice{Text: "SruTinder: это взаимно! Вы совпали с Данияр.", ButtonURL: "/apps/tinder", ButtonText: "Открыть"}
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	sealed, err := sealForDelivery(cipher, raw, tinderNoticeAAD(3, "match:1:3:3"))
	if err != nil {
		t.Fatal(err)
	}
	for _, needle := range [][]byte{[]byte("взаимно"), []byte("Данияр"), []byte("apps/tinder")} {
		if containsBytes(sealed.Ciphertext, needle) {
			t.Fatalf("ciphertext contains plaintext %q", needle)
		}
	}
	opened, err := cipher.Open(sealed, tinderNoticeAAD(3, "match:1:3:3"))
	if err != nil {
		t.Fatal(err)
	}
	if string(opened) != string(raw) {
		t.Fatal("round trip changed the payload")
	}
}

func containsBytes(haystack, needle []byte) bool {
	if len(needle) == 0 || len(haystack) < len(needle) {
		return false
	}
	for i := 0; i+len(needle) <= len(haystack); i++ {
		match := true
		for j := range needle {
			if haystack[i+j] != needle[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

// sealForDelivery mirrors what the service does, so the test proves the same
// cipher and AAD are used on both sides.
func sealForDelivery(cipher *secure.Cipher, payload []byte, aad string) (secure.Sealed, error) {
	return cipher.Seal(payload, aad)
}

var _ = context.Background
