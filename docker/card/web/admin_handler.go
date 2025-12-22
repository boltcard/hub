package web

import (
	"card/db"
	"net/http"
	"strings"

	log "github.com/sirupsen/logrus"
)

func (app *App) CreateHandler_Admin() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		request := r.RequestURI

		//log.Info("CreateHandler_Admin handler with request uri : " + request)

		// prevent caching
		w.Header().Add("Cache-Control", "no-cache, no-store")

		if request == "/admin/register/" {
			Register2(app.db_conn, w, r)
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
			Login2(app.db_conn, w, r)
			return
		}

		// detect if a session cookie exists
		c, err := r.Cookie("admin_session_token")
		if err != nil {
			if err == http.ErrNoCookie {
				//redirect to "login" page
				http.Redirect(w, r, "/admin/login/", http.StatusSeeOther)
				return
			}
			log.Info("admin_session_token error : ", err.Error())
			Blank(w, nil)
			return
		}

		// validate the session cookie
		sessionToken := c.Value
		adminSessionToken := db.Db_get_setting(app.db_conn, "admin_session_token")

		if sessionToken != adminSessionToken {
			ClearAdminSessionToken(w)
			//redirect to "login" page
			http.Redirect(w, r, "/admin/login/", http.StatusSeeOther)
			return
		}

		if strings.HasSuffix(request, ".js") || strings.HasSuffix(request, ".css") ||
			strings.HasSuffix(request, ".png") || strings.HasSuffix(request, ".jpg") ||
			strings.HasSuffix(request, ".map") {
			RenderStaticContent(w, request)
			return
		}

		if request == "/admin/" {
			Admin_Index(w, r)
		}

		if strings.HasPrefix(request, "/admin/phoenix/") {
			Admin_Phoenix(app.db_conn, w, r)
		}

		if strings.HasPrefix(request, "/admin/cards/") {
			Admin_Cards(app.db_conn, w, r)
		}

		if strings.HasPrefix(request, "/admin/settings/") {
			Admin_Settings(app.db_conn, w, r)
		}

		if strings.HasPrefix(request, "/admin/about/") {
			Admin_About(app.db_conn, w, r)
		}

		Blank(w, r)
	}
}
