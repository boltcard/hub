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

	ClearAdmin2SessionToken(w)

	// handle postback
	if r.Method == "POST" {
		r.ParseForm()

		passwordStr := r.Form.Get("password")
		passwordHashStr := GetPwHash(db_conn, passwordStr)

		// check password
		if db.Db_get_setting(db_conn, "admin2_password_hash") == passwordHashStr {
			sessionToken := util.Random_hex()

			db.Db_set_setting(db_conn, "admin2_session_token", sessionToken)

			http.SetCookie(w, &http.Cookie{
				Name:    "admin2_session_token",
				Value:   sessionToken,
				Path:    "/admin2/",
				Expires: time.Now().Add(24 * time.Hour),
			})

			// TODO: add 2FA

			//renderContent(w, request)
			http.Redirect(w, r, "/admin2/", http.StatusSeeOther)
			return
		}

		// failed login
		log.Warn("a failed login happened")
		http.Redirect(w, r, "/admin2/login/", http.StatusSeeOther)
		return
	}

	// return page for user to login as admin
	RenderHtmlFromTemplate(w, "/admin2/login/index.html", nil)
}
