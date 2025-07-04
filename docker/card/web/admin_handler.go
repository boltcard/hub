package web

import (
	"card/db"
	"net/http"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

func (app *App) CreateHandler_Admin() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		request := r.RequestURI

		// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Cache-Control
		w.Header().Add("Cache-Control", "no-cache")

		// items to return without an authenticated session
		if strings.HasSuffix(request, ".js") || strings.HasSuffix(request, ".css") ||
			strings.HasSuffix(request, ".png") || strings.HasSuffix(request, ".jpg") ||
			strings.HasSuffix(request, ".map") {
			RenderStaticContent(w, request)
			return
		}

		if request == "/admin/register/" {
			Register(app.db_conn, w, r)
			return
		}

		// detect if an admin password has been set
		if db.Db_get_setting(app.db_conn, "admin_password_hash") == "" {
			// https://freshman.tech/snippets/go/http-redirect/
			//redirect to "register" page
			http.Redirect(w, r, "/admin/register/", http.StatusSeeOther)
			return
		}

		if request == "/admin/login/" {
			Login(app.db_conn, w, r)
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
		adminSessionToken := db.Db_get_setting(app.db_conn, "session_token")

		if sessionToken != adminSessionToken {
			ClearAdminSessionToken(w)
			//redirect to "login" page
			http.Redirect(w, r, "/admin/login/", http.StatusSeeOther)
			return
		}

		log.Info("request: ", request)

		if request == "/admin/" {
			Index(w, r)
		}

		if strings.HasPrefix(request, "/admin/payments-in/") {
			PaymentsIn(app.db_conn, w, r)
		}

		if strings.HasPrefix(request, "/admin/payments-out/") {
			PaymentsOut(app.db_conn, w, r)
		}

		if strings.HasPrefix(request, "/admin/bolt-card/") {
			BoltCard(app.db_conn, w, r)
		}

		Blank(w, r)
	}
}

func ClearAdminSessionToken(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:    "session_token",
		Value:   "",
		Path:    "/admin/",
		Expires: time.Now(),
	})
}
