package util

import (
	"encoding/base64"
	"errors"
	"testing"
)

func TestConvertPaymentHash_ValidHex(t *testing.T) {
	got := ConvertPaymentHash("00ff10")
	want := []int{0, 255, 16}
	if len(got) != len(want) {
		t.Fatalf("expected %d ints, got %d (%v)", len(want), len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("at index %d expected %d, got %d", i, want[i], got[i])
		}
	}
}

func TestConvertPaymentHash_Empty(t *testing.T) {
	got := ConvertPaymentHash("")
	if len(got) != 0 {
		t.Fatalf("expected empty slice, got %v", got)
	}
}

func TestConvertPaymentHash_InvalidHex(t *testing.T) {
	got := ConvertPaymentHash("zzzz")
	if got != nil {
		t.Fatalf("expected nil for invalid hex, got %v", got)
	}
}

func TestConvertPaymentHash_OddLength(t *testing.T) {
	// odd-length strings are not valid hex
	got := ConvertPaymentHash("abc")
	if got != nil {
		t.Fatalf("expected nil for odd-length hex, got %v", got)
	}
}

func TestRandomHex_LengthAndHex(t *testing.T) {
	h := Random_hex()
	// 16 random bytes -> 32 hex characters
	if len(h) != 32 {
		t.Fatalf("expected 32 hex chars, got %d (%q)", len(h), h)
	}
	for _, c := range h {
		isHex := (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')
		if !isHex {
			t.Fatalf("non-hex character %q in %q", c, h)
		}
	}
}

func TestRandomHex_Unique(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		h := Random_hex()
		if seen[h] {
			t.Fatalf("duplicate Random_hex value: %q", h)
		}
		seen[h] = true
	}
}

func TestQrPngBase64Encode_ProducesPng(t *testing.T) {
	encoded := QrPngBase64Encode("lnurl1test")
	if encoded == "" {
		t.Fatal("expected non-empty base64 output")
	}

	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("output is not valid base64: %v", err)
	}

	// PNG files start with the 8-byte signature 89 50 4E 47 0D 0A 1A 0A
	pngHeader := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	if len(raw) < len(pngHeader) {
		t.Fatalf("decoded output too short to be a PNG (%d bytes)", len(raw))
	}
	for i, b := range pngHeader {
		if raw[i] != b {
			t.Fatalf("decoded output is not a PNG (byte %d = 0x%02X, want 0x%02X)", i, raw[i], b)
		}
	}
}

func TestCheckAndLog_NilDoesNotPanic(t *testing.T) {
	// Should be a no-op and must not panic.
	CheckAndLog(nil)
	CheckAndLog(errors.New("logged error"))
}

func TestCheckAndPanic_NilDoesNotPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("CheckAndPanic(nil) panicked: %v", r)
		}
	}()
	CheckAndPanic(nil)
}

func TestCheckAndPanic_NonNilPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("CheckAndPanic(err) did not panic")
		}
	}()
	CheckAndPanic(errors.New("boom"))
}
