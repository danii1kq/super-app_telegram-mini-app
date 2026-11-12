package notes

import "testing"

// The AAD must bind a note to its owner: a ciphertext moved to another note or
// another owner has to fail, which is what these strings guarantee.
func TestAADBindsNoteToOwner(t *testing.T) {
	if titleAAD(1, 7) == titleAAD(1, 8) {
		t.Fatal("title AAD must differ per owner")
	}
	if bodyAAD(1, 7) == bodyAAD(2, 7) {
		t.Fatal("body AAD must differ per note")
	}
	if titleAAD(1, 7) == bodyAAD(1, 7) {
		t.Fatal("title and body must not share an AAD")
	}
}

func TestItoa(t *testing.T) {
	cases := map[int64]string{
		0:          "0",
		7:          "7",
		42:         "42",
		1000000:    "1000000",
		-15:        "-15",
		9223372036854775807: "9223372036854775807",
	}
	for input, want := range cases {
		if got := itoa(input); got != want {
			t.Fatalf("itoa(%d) = %q, want %q", input, got, want)
		}
	}
}
