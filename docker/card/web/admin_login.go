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
		storedHash := db.Db_get_setting(db_conn, "admin_password_hash")

		// check password (supports both bcrypt and legacy SHA256)
		passwordValid := false
		if isBcryptHash(storedHash) {
			passwordValid = CheckPassword(passwordStr, storedHash)
		} else {
			// legacy SHA256 check
			passwordHashStr := GetPwHash(db_conn, passwordStr)
			if storedHash == passwordHashStr {
				passwordValid = true
				// migrate to bcrypt
				newHash, err := HashPassword(passwordStr)
				if err == nil {
					db.Db_set_setting(db_conn, "admin_password_hash", newHash)
					log.Info("migrated admin password to bcrypt")
				}
			}
		}

		if passwordValid {
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
