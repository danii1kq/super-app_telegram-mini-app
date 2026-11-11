package integration

import "testing"

// The content hash is what makes an idempotency key safe: the same key with the
// same content is a retry, with different content it is a conflict.
func TestHashContentIsStableAndSensitive(t *testing.T) {
	first, err := hashContent(map[string]string{"order": "A-1", "title": "Курсовая", "due": "2026-11-20"})
	if err != nil {
		t.Fatalf("hashContent: %v", err)
	}
	second, err := hashContent(map[string]string{"order": "A-1", "title": "Курсовая", "due": "2026-11-20"})
	if err != nil {
		t.Fatalf("hashContent: %v", err)
	}
	if first != second {
		t.Fatal("the same content must produce the same hash")
	}

	changed, err := hashContent(map[string]string{"order": "A-1", "title": "Курсовая", "due": "2026-11-21"})
	if err != nil {
		t.Fatalf("hashContent: %v", err)
	}
	if first == changed {
		t.Fatal("a different date must change the hash")
	}
	if len(first) != 64 {
		t.Fatalf("expected a hex sha256, got %q", first)
	}
}
