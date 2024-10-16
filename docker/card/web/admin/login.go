package admin

import (
	"card/db"
	"card/util"
	"card/web"
	"time"

	"net/http"

	log "github.com/sirupsen/logrus"
)

func Login(w http.ResponseWriter, r *http.Request) {

	web.ClearSessionToken(w)

	// handle postback
	if r.Method == "POST" {
		r.ParseForm()

		passwordStr := r.Form.Get("password")
		passwordHashStr := web.GetPwHash(passwordStr)

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
		log.Warn("a failed login happened")
		http.Redirect(w, r, "/admin/login/", http.StatusSeeOther)
		return
	}

	// return page for user to login as admin
	web.RenderHtmlFromTemplate(w, "/dist/pages/admin/login/index.html", nil)
}
