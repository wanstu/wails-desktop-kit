package secureconfig

import (
	"encoding/base64"
	"encoding/hex"
	"testing"
)

func TestRandomHexMatchesOpenSSLStyleLength(t *testing.T) {
	value, err := RandomHex(32)
	if err != nil {
		t.Fatal(err)
	}
	if len(value) != 64 {
		t.Fatalf("RandomHex(32) length = %d, want 64", len(value))
	}
	decoded, err := hex.DecodeString(value)
	if err != nil {
		t.Fatalf("RandomHex returned non-hex text: %v", err)
	}
	if len(decoded) != 32 {
		t.Fatalf("decoded length = %d, want 32", len(decoded))
	}
}

func TestRandomBase64URLRoundTripLength(t *testing.T) {
	value, err := RandomBase64URL(32)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		t.Fatalf("RandomBase64URL returned invalid base64url: %v", err)
	}
	if len(decoded) != 32 {
		t.Fatalf("decoded length = %d, want 32", len(decoded))
	}
}

func TestRandomSecretHexUses256Bits(t *testing.T) {
	first, err := RandomSecretHex()
	if err != nil {
		t.Fatal(err)
	}
	second, err := RandomSecretHex()
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 64 || len(second) != 64 {
		t.Fatalf("unexpected secret lengths: %d, %d", len(first), len(second))
	}
	if first == second {
		t.Fatal("two independently generated random secrets unexpectedly matched")
	}
}

func TestRandomRejectsNonPositiveSize(t *testing.T) {
	for _, size := range []int{0, -1} {
		if _, err := RandomBytes(size); err == nil {
			t.Fatalf("RandomBytes(%d) unexpectedly succeeded", size)
		}
		if _, err := RandomHex(size); err == nil {
			t.Fatalf("RandomHex(%d) unexpectedly succeeded", size)
		}
		if _, err := RandomBase64URL(size); err == nil {
			t.Fatalf("RandomBase64URL(%d) unexpectedly succeeded", size)
		}
	}
}
