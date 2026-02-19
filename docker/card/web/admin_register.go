package web

import (
	"card/db"
	"database/sql"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
)

func Register2(db_conn *sql.DB, w http.ResponseWriter, r *http.Request) {

	ClearAdminSessionToken(w)

	// this protects from setting a new admin_password_hash when it has already been set
	if db.Db_get_setting(db_conn, "admin_password_hash") != "" {
		//redirect to "login" page
		http.Redirect(w, r, "/admin/login/", http.StatusSeeOther)
		return
	}

	// handle postback
	if r.Method == "POST" {
		r.ParseForm()

		passwordStr := r.Form.Get("password")
		passwordHashStr, err := HashPassword(passwordStr)
		if err != nil {
			log.Error("failed to hash password: ", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		// TODO: check that password has >128 bit entropy to mitigate brute force attacks

		db.Db_set_setting(db_conn, "admin_password_hash", passwordHashStr)

		// TODO: redirect to 2FA setup

		//redirect to "login" page
		http.Redirect(w, r, "/admin/login/", http.StatusSeeOther)
		return
	}

	// return page for user to set an admin password
	RenderHtmlFromTemplate(w, "/admin/register/index.html", nil)
}

func ClearAdminSessionToken(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:    "admin_session_token",
		Value:   "",
		Path:    "/admin/",
		Expires: time.Now(),
	})
}
