package crypto

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"testing"
)

func TestAesDecrypt_RoundTrip(t *testing.T) {
	key := []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,
		0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f}
	plaintext := []byte{0xDE, 0xAD, 0xBE, 0xEF, 0x01, 0x02, 0x03, 0x04,
		0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C}

	// encrypt with AES-CBC and zero IV (matching Aes_decrypt's assumption)
	block, err := aes.NewCipher(key)
	if err != nil {
		t.Fatal(err)
	}
	iv := make([]byte, 16)
	ciphertext := make([]byte, 16)
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(ciphertext, plaintext)

	// decrypt and verify round trip
	result, err := Aes_decrypt(key, ciphertext)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Equal(result, plaintext) {
		t.Fatalf("decrypted %x, want %x", result, plaintext)
	}
}

func TestAesDecrypt_ZeroPlaintext(t *testing.T) {
	key := []byte{0x2b, 0x7e, 0x15, 0x16, 0x28, 0xae, 0xd2, 0xa6,
		0xab, 0xf7, 0x15, 0x88, 0x09, 0xcf, 0x4f, 0x3c}
	plaintext := make([]byte, 16) // all zeros

	block, _ := aes.NewCipher(key)
	iv := make([]byte, 16)
	ciphertext := make([]byte, 16)
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(ciphertext, plaintext)

	result, err := Aes_decrypt(key, ciphertext)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Equal(result, plaintext) {
		t.Fatalf("decrypted %x, want %x", result, plaintext)
	}
}

func TestAesDecrypt_WrongKeyLength(t *testing.T) {
	badKey := []byte{0x00, 0x01, 0x02} // too short
	ciphertext := make([]byte, 16)

	_, err := Aes_decrypt(badKey, ciphertext)
	if err == nil {
		t.Fatal("expected error for invalid key length")
	}
}
