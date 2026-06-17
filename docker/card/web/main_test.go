package web

import (
	"os"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

// TestMain lowers the bcrypt work factor for the whole web test package.
// Production keeps bcrypt.DefaultCost (see HashPassword in util.go); the
// admin-login and 2FA recovery-code tests hash dozens of values per run, and
// at DefaultCost that hashing — amplified by the race detector — dominated
// CI's `go test -race` step (~3 min). MinCost keeps the hashing path fully
// exercised while making it roughly 64x cheaper.
func TestMain(m *testing.M) {
	bcryptCost = bcrypt.MinCost
	os.Exit(m.Run())
}
