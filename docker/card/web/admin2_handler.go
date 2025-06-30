package web

import (
	"card/db"
	"net/http"
	"strings"

	log "github.com/sirupsen/logrus"
)

func (app *App) CreateHandler_Admin2() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		request := r.RequestURI

		log.Info("CreateHandler_Admin2 handler with request uri : " + request)

		// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Cache-Control
		w.Header().Add("Cache-Control", "no-cache")

		//HACK: these need authenticating also
		// items to return without an authenticated session
		if strings.HasSuffix(request, ".js") || strings.HasSuffix(request, ".css") ||
			strings.HasSuffix(request, ".png") || strings.HasSuffix(request, ".jpg") ||
			strings.HasSuffix(request, ".map") {
			RenderStaticContent(w, request)
			return
		}

		if request == "/admin2/register/" {
			Register2(app.db_conn, w, r)
			return
		}

		// detect if an admin password has been set
		if db.Db_get_setting(app.db_conn, "admin2_password_hash") == "" {
			// https://freshman.tech/snippets/go/http-redirect/
			//redirect to "register" page
			http.Redirect(w, r, "/admin2/register/", http.StatusSeeOther)
			return
		}

		if request == "/admin2/login/" {
			Login2(app.db_conn, w, r)
			return
		}

		// detect if a session cookie exists
		c, err := r.Cookie("admin2_session_token")
		if err != nil {
			if err == http.ErrNoCookie {
				//redirect to "login" page
				http.Redirect(w, r, "/admin2/login/", http.StatusSeeOther)
				return
			}
			log.Info("admin2_session_token error : ", err.Error())
			Blank(w, nil)
			return
		}

		// validate the session cookie
		sessionToken := c.Value
		adminSessionToken := db.Db_get_setting(app.db_conn, "admin2_session_token")

		if sessionToken != adminSessionToken {
			ClearAdmin2SessionToken(w)
			//redirect to "login" page
			http.Redirect(w, r, "/admin2/login/", http.StatusSeeOther)
			return
		}

		log.Info("request: ", request)

		if request == "/admin2/" {
			Admin2_Index(w, r)
		}

		if strings.HasPrefix(request, "/admin2/phoenix/") {
			Admin2_Phoenix(app.db_conn, w, r)
		}

		if strings.HasPrefix(request, "/admin2/cards/") {
			Admin2_Cards(app.db_conn, w, r)
		}

		if strings.HasPrefix(request, "/admin2/settings/") {
			Admin2_Settings(app.db_conn, w, r)
		}

		if strings.HasPrefix(request, "/admin2/about/") {
			Admin2_About(app.db_conn, w, r)
		}

		Blank(w, r)
	}
}
