package web

import (
	"net/http"
	"time"
)

func ClearAdminSessionToken(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "admin_session_token",
		Value:    "",
		Path:     "/admin/",
		Expires:  time.Now(),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})
}
