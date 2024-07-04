package web

import (
	"card/db"
	"net/http"
)

func Register(w http.ResponseWriter, r *http.Request) {

	clearSessionToken(w)

	// this protects from setting a new admin_password_hash when it has already been set
	if db.Db_get_setting("admin_password_hash") != "" {
		//redirect to "login" page
		http.Redirect(w, r, "/admin/login/", http.StatusSeeOther)
		return
	}

	// handle postback
	if r.Method == "POST" {
		r.ParseForm()

		passwordStr := r.Form.Get("password")
		passwordHashStr := getPwHash(passwordStr)

		db.Db_set_setting("admin_password_hash", passwordHashStr)

		// TODO: redirect to 2FA setup

		//redirect to "login" page
		http.Redirect(w, r, "/admin/login/", http.StatusSeeOther)
		return
	}

	// return page for user to set an admin password
	renderHtmlFromTemplate(w, "/dist/pages/admin/register/index.html", nil)
}
