package web

import (
	"card/db"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"time"
)

func ClearSessionToken(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:    "session_token",
		Value:   "",
		Path:    "/admin/",
		Expires: time.Now(),
	})
}

func GetPwHash(passwordStr string) (passwordHashStr string) {
	passwordSalt := db.Db_get_setting("admin_password_salt")

	hasher := sha256.New()
	hasher.Write([]byte(passwordSalt))
	hasher.Write([]byte(passwordStr))
	passwordHash := hasher.Sum(nil)
	passwordHashStr = hex.EncodeToString(passwordHash)

	return passwordHashStr
}
