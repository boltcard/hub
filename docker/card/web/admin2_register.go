package web

import (
	"card/db"
	"database/sql"
	"net/http"
	"time"
)

func Register2(db_conn *sql.DB, w http.ResponseWriter, r *http.Request) {

	ClearAdmin2SessionToken(w)

	// this protects from setting a new admin_password_hash when it has already been set
	if db.Db_get_setting(db_conn, "admin2_password_hash") != "" {
		//redirect to "login" page
		http.Redirect(w, r, "/admin2/login/", http.StatusSeeOther)
		return
	}

	// handle postback
	if r.Method == "POST" {
		r.ParseForm()

		passwordStr := r.Form.Get("password")
		passwordHashStr := GetPwHash(db_conn, passwordStr)

		// TODO: check that password has >128 bit entropy to mitigate brute force attacks

		db.Db_set_setting(db_conn, "admin2_password_hash", passwordHashStr)

		// TODO: redirect to 2FA setup

		//redirect to "login" page
		http.Redirect(w, r, "/admin2/login/", http.StatusSeeOther)
		return
	}

	// return page for user to set an admin password
	RenderHtmlFromTemplate(w, "/admin2/register/index.html", nil)
}

func ClearAdmin2SessionToken(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:    "admin2_session_token",
		Value:   "",
		Path:    "/admin2/",
		Expires: time.Now(),
	})
}
