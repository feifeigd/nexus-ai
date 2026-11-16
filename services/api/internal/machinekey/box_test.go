package machinekey

import (
	"strings"
	"testing"
)

func TestBoxRoundTripAndRandomNonce(t *testing.T) {
	box, err := NewBox("test-secret-123")
	if err != nil {
		t.Fatalf("new box: %v", err)
	}
	const plaintext = "-----BEGIN OPENSSH PRIVATE KEY-----\nsecret\n-----END OPENSSH PRIVATE KEY-----"

	first, err := box.Seal(plaintext)
	if err != nil {
		t.Fatalf("seal first value: %v", err)
	}
	second, err := box.Seal(plaintext)
	if err != nil {
		t.Fatalf("seal second value: %v", err)
	}
	if !IsEncrypted(first) || strings.Contains(first, plaintext) || strings.Contains(first, "secret") {
		t.Fatalf("ciphertext exposes plaintext: %q", first)
	}
	if first == second {
		t.Fatal("expected random nonces to produce different ciphertexts")
	}
	opened, err := box.Open(first)
	if err != nil || opened != plaintext {
		t.Fatalf("open round trip: opened=%q err=%v", opened, err)
	}
}

func TestBoxRejectsTamperingAndWrongSecret(t *testing.T) {
	box, err := NewBox("test-secret-123")
	if err != nil {
		t.Fatalf("new box: %v", err)
	}
	sealed, err := box.Seal("private-key")
	if err != nil {
		t.Fatalf("seal: %v", err)
	}

	replacement := "A"
	if strings.HasSuffix(sealed, replacement) {
		replacement = "B"
	}
	tampered := sealed[:len(sealed)-1] + replacement
	if _, err := box.Open(tampered); err == nil {
		t.Fatal("expected tampered ciphertext to be rejected")
	}
	other, err := NewBox("another-secret")
	if err != nil {
		t.Fatalf("new other box: %v", err)
	}
	if _, err := other.Open(sealed); err == nil {
		t.Fatal("expected a different secret to fail authentication")
	}
	if _, err := box.Open("legacy-plaintext"); err == nil {
		t.Fatal("expected a value without a ciphertext version to be rejected")
	}
}

func TestNewBoxRequiresSecret(t *testing.T) {
	if _, err := NewBox("  "); err == nil {
		t.Fatal("expected an empty secret to be rejected")
	}
}
