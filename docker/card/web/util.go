package web

import (
	"card/db"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// GetPwHash computes the legacy SHA256(salt+password) hash.
// Only used to verify old passwords during login, which then migrates
// the stored hash to bcrypt. See admin_login.go.
func GetPwHash(db_conn *sql.DB, passwordStr string) (passwordHashStr string) {
	passwordSalt := db.Db_get_setting(db_conn, "admin_password_salt")

	hasher := sha256.New()
	hasher.Write([]byte(passwordSalt))
	hasher.Write([]byte(passwordStr))
	passwordHash := hasher.Sum(nil)
	passwordHashStr = hex.EncodeToString(passwordHash)

	return passwordHashStr
}

func HashPassword(passwordStr string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(passwordStr), bcrypt.DefaultCost)
	return string(hash), err
}

func CheckPassword(passwordStr string, hashStr string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hashStr), []byte(passwordStr)) == nil
}

func isBcryptHash(hash string) bool {
	return strings.HasPrefix(hash, "$2a$") || strings.HasPrefix(hash, "$2b$")
}
