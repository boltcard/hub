package web

import (
	"strings"
	"testing"
	"time"

	"github.com/pquerna/otp/totp"
)

func TestNewTotpKey_ProducesValidatableSecret(t *testing.T) {
	secret, url, err := newTotpKey("hub.example.com")
	if err != nil {
		t.Fatal(err)
	}
	if secret == "" {
		t.Fatal("expected non-empty secret")
	}
	if !strings.HasPrefix(url, "otpauth://totp/") {
		t.Fatalf("expected otpauth URI, got %q", url)
	}

	code, err := totp.GenerateCode(secret, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if !validateTotpCode(secret, code) {
		t.Fatal("freshly generated code should validate")
	}
	// A code generated for 24h ago must not validate now. Skip the
	// astronomically unlikely case where it equals the current code, so the
	// test is deterministic rather than flaky.
	past, err := totp.GenerateCode(secret, time.Now().Add(-24*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if past != code && validateTotpCode(secret, past) {
		t.Fatal("a code from 24h ago should not validate now")
	}
}

func TestGenerateRecoveryCodes_HashesMatch(t *testing.T) {
	plain, hashes, err := generateRecoveryCodes(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(plain) != 10 || len(hashes) != 10 {
		t.Fatalf("expected 10 codes and 10 hashes, got %d and %d", len(plain), len(hashes))
	}
	for i := range plain {
		if !CheckPassword(plain[i], hashes[i]) {
			t.Fatalf("hash %d does not verify against its plaintext", i)
		}
	}
	seen := map[string]bool{}
	for _, c := range plain {
		if seen[c] {
			t.Fatalf("duplicate recovery code %q", c)
		}
		seen[c] = true
	}
}

func TestMatchRecoveryCode(t *testing.T) {
	plain, hashes, err := generateRecoveryCodes(3)
	if err != nil {
		t.Fatal(err)
	}
	idx, ok := matchRecoveryCode(plain[1], hashes)
	if !ok || idx != 1 {
		t.Fatalf("expected match at index 1, got idx=%d ok=%v", idx, ok)
	}
	if _, ok := matchRecoveryCode("nope-not-a-code", hashes); ok {
		t.Fatal("a non-code should not match")
	}
}
