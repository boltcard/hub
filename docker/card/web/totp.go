package web

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/pquerna/otp/totp"
)

// newTotpKey generates a fresh TOTP secret for the admin. accountName is the
// label shown in the authenticator app (we pass host_domain). The issuer is
// "Boltcard Hub" (one token) to avoid wrong-logo brand matching in Authy.
// Returns the base32 secret and the otpauth:// provisioning URI.
func newTotpKey(accountName string) (secret string, url string, err error) {
	key, err := totp.Generate(totp.GenerateOpts{
		// "Boltcard" as a single token (not "Bolt Card") so authenticator apps
		// that pick a logo by matching the issuer (e.g. Authy) don't false-match
		// the unrelated "Bolt" brand and suggest the wrong logo.
		Issuer:      "Boltcard Hub",
		AccountName: accountName,
	})
	if err != nil {
		return "", "", err
	}
	return key.Secret(), key.URL(), nil
}

// validateTotpCode checks a 6-digit code against the secret. totp.Validate
// already applies a ±1 time-step skew to tolerate clock drift.
func validateTotpCode(secret, code string) bool {
	return totp.Validate(code, secret)
}

// generateRecoveryCodes returns `count` single-use recovery codes (16 hex
// chars each, 64 bits of entropy) plus their bcrypt hashes. Only the hashes
// are persisted; the plaintext is shown to the admin exactly once.
func generateRecoveryCodes(count int) (plain []string, hashes []string, err error) {
	for range count {
		b := make([]byte, 8)
		if _, err = rand.Read(b); err != nil {
			return nil, nil, err
		}
		code := hex.EncodeToString(b)
		h, herr := HashPassword(code)
		if herr != nil {
			return nil, nil, herr
		}
		plain = append(plain, code)
		hashes = append(hashes, h)
	}
	return plain, hashes, nil
}

// matchRecoveryCode returns the index of the first hash that verifies against
// code, or ok=false if none match.
func matchRecoveryCode(code string, hashes []string) (int, bool) {
	for i, h := range hashes {
		if CheckPassword(code, h) {
			return i, true
		}
	}
	return -1, false
}
