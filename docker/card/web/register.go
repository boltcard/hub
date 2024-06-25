package web

import (
	"card/db"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
)

func getPwHash(passwordStr string) (passwordHashStr string) {
	passwordSalt := db.Db_get_setting("admin_password_salt")

	hasher := sha256.New()
	hasher.Write([]byte(passwordSalt))
	hasher.Write([]byte(passwordStr))
	passwordHash := hasher.Sum(nil)
	passwordHashStr = hex.EncodeToString(passwordHash)

	return passwordHashStr
}

func Register(w http.ResponseWriter, r *http.Request) {

	// handle postback
	if r.Method == "POST" {
		r.ParseForm()

		passwordStr := r.Form.Get("password")
		passwordHashStr := getPwHash(passwordStr)

		// double check that we are not overwriting an admin_password_hash
		if db.Db_get_setting("admin_password_hash") == "" {
			db.Db_set_setting("admin_password_hash", passwordHashStr)
		}

		// TODO: redirect to 2FA setup

		//redirect to "login" page
		http.Redirect(w, r, "/admin/login/", http.StatusSeeOther)
		return
	}

	// return page for user to set an admin password
	renderContent(w, r)
}
