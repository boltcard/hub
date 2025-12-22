package web

import (
	"card/db"
	"card/util"
	"database/sql"
	"time"

	"net/http"

	log "github.com/sirupsen/logrus"
)

func Login2(db_conn *sql.DB, w http.ResponseWriter, r *http.Request) {

	ClearAdminSessionToken(w)

	// handle postback
	if r.Method == "POST" {
		r.ParseForm()

		passwordStr := r.Form.Get("password")
		passwordHashStr := GetPwHash(db_conn, passwordStr)

		// check password
		if db.Db_get_setting(db_conn, "admin_password_hash") == passwordHashStr {
			sessionToken := util.Random_hex()

			db.Db_set_setting(db_conn, "admin_session_token", sessionToken)

			http.SetCookie(w, &http.Cookie{
				Name:    "admin_session_token",
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
	RenderHtmlFromTemplate(w, "/admin/login/index.html", nil)
}
