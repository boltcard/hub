package web

import (
	"card/db"
	"card/util"
	"time"

	"net/http"
)

func Login(w http.ResponseWriter, r *http.Request) {

	request := r.RequestURI

	http.SetCookie(w, &http.Cookie{
		Name:    "session_token",
		Value:   "",
		Path:    "/admin/",
		Expires: time.Now(),
	})

	// handle postback
	if r.Method == "POST" {
		r.ParseForm()

		passwordStr := r.Form.Get("password")
		passwordHashStr := getPwHash(passwordStr)

		// TODO: add rate limiting

		// check password
		if db.Db_get_setting("admin_password_hash") == passwordHashStr {
			sessionToken := util.Random_hex()

			db.Db_set_setting("session_token", sessionToken)

			http.SetCookie(w, &http.Cookie{
				Name:    "session_token",
				Value:   sessionToken,
				Path:    "/admin/",
				Expires: time.Now().Add(24 * time.Hour),
			})

			// TODO: add 2FA

			//renderContent(w, request)
			http.Redirect(w, r, "/admin/", http.StatusSeeOther)
			return
		}

		// failed login
		http.Redirect(w, r, "/admin/login/", http.StatusSeeOther)
		return
	}

	// return page for user to login as admin
	renderContent(w, request)
}
