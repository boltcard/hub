package web

import (
	"card/db"
	"net/http"
	"strings"

	log "github.com/sirupsen/logrus"
)

func Admin(w http.ResponseWriter, r *http.Request) {
	request := r.RequestURI

	// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Cache-Control
	w.Header().Add("Cache-Control", "no-cache")

	// items to return without an authenticated session
	if strings.HasSuffix(request, ".js") || strings.HasSuffix(request, ".css") ||
		strings.HasSuffix(request, ".png") || strings.HasSuffix(request, ".jpg") ||
		strings.HasSuffix(request, ".map") {
		renderContent(w, request)
		return
	}

	if request == "/admin/register/" {
		Register(w, r)
		return
	}

	// detect if an admin password has been set
	if db.Db_get_setting("admin_password_hash") == "" {
		// https://freshman.tech/snippets/go/http-redirect/
		//redirect to "register" page
		http.Redirect(w, r, "/admin/register/", http.StatusSeeOther)
		return
	}

	if request == "/admin/login/" {
		Login(w, r)
		return
	}

	// detect if a session cookie exists
	c, err := r.Cookie("session_token")
	if err != nil {
		if err == http.ErrNoCookie {
			//redirect to "login" page
			http.Redirect(w, r, "/admin/login/", http.StatusSeeOther)
			return
		}
		log.Info("session_token error : ", err.Error())
		Blank(w, nil)
		return
	}

	// validate the session cookie
	sessionToken := c.Value
	adminSessionToken := db.Db_get_setting("session_token")

	if sessionToken != adminSessionToken {
		clearSessionToken(w)
		//redirect to "login" page
		http.Redirect(w, r, "/admin/login/", http.StatusSeeOther)
		return
	}

	// if request == "/admin/" {
	// 	Dashboard(w, r)
	// }

	if request == "/admin/payments/" {
		Payments(w, r)
	}

	Blank(w, r)
	//renderContent(w, request)
}
