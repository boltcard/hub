package web

import (
	"card/db"
	"crypto/subtle"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

func (app *App) CreateHandler_Admin() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		request := r.RequestURI

		// prevent caching
		w.Header().Add("Cache-Control", "no-cache, no-store")

		// Static assets — serve without auth (SPA assets first, then legacy)
		if strings.HasSuffix(request, ".js") || strings.HasSuffix(request, ".css") ||
			strings.HasSuffix(request, ".png") || strings.HasSuffix(request, ".jpg") ||
			strings.HasSuffix(request, ".map") || strings.HasSuffix(request, ".woff2") ||
			strings.HasSuffix(request, ".svg") || strings.HasSuffix(request, ".ico") {
			spaPath := strings.Replace(request, "/admin/", "/admin/spa/", 1)
			if fileExists("/web-content" + spaPath) {
				RenderStaticContent(w, spaPath)
			} else {
				RenderStaticContent(w, request)
			}
			return
		}

		// Serve React SPA for all non-static admin paths.
		// The SPA handles auth via /admin/api/auth/check.
		if fileExists("/web-content/admin/spa/index.html") {
			serveSpaIndex(w, r)
			return
		}

		// Fallback: legacy Go template admin UI (when SPA not built)
		if request == "/admin/register/" {
			Register2(app.db_conn, w, r)
			return
		}

		if db.Db_get_setting(app.db_conn, "admin_password_hash") == "" {
			http.Redirect(w, r, "/admin/register/", http.StatusSeeOther)
			return
		}

		if request == "/admin/login/" {
			Login2(app.db_conn, w, r)
			return
		}

		c, err := r.Cookie("admin_session_token")
		if err != nil {
			if err == http.ErrNoCookie {
				http.Redirect(w, r, "/admin/login/", http.StatusSeeOther)
				return
			}
			log.Info("admin_session_token error : ", err.Error())
			Blank(w, nil)
			return
		}

		sessionToken := c.Value
		adminSessionToken := db.Db_get_setting(app.db_conn, "admin_session_token")

		if subtle.ConstantTimeCompare([]byte(sessionToken), []byte(adminSessionToken)) != 1 {
			ClearAdminSessionToken(w)
			http.Redirect(w, r, "/admin/login/", http.StatusSeeOther)
			return
		}

		sessionCreatedStr := db.Db_get_setting(app.db_conn, "admin_session_created")
		if sessionCreatedStr != "" {
			sessionCreated, err := strconv.ParseInt(sessionCreatedStr, 10, 64)
			if err != nil || time.Now().Unix()-sessionCreated > 24*60*60 {
				ClearAdminSessionToken(w)
				http.Redirect(w, r, "/admin/login/", http.StatusSeeOther)
				return
			}
		}

		switch {
		case request == "/admin/":
			Admin_Index(app.db_conn, w, r)
		case strings.HasPrefix(request, "/admin/phoenix/"):
			Admin_Phoenix(app.db_conn, w, r)
		case strings.HasPrefix(request, "/admin/cards/"):
			Admin_Cards(app.db_conn, w, r)
		case strings.HasPrefix(request, "/admin/settings/"):
			Admin_Settings(app.db_conn, w, r)
		case strings.HasPrefix(request, "/admin/about/"):
			Admin_About(app.db_conn, w, r)
		case strings.HasPrefix(request, "/admin/database/"):
			Admin_Database(app.db_conn, w, r)
		default:
			Blank(w, r)
		}
	}
}

func serveSpaIndex(w http.ResponseWriter, r *http.Request) {
	spaIndex := "/web-content/admin/spa/index.html"

	content, err := os.ReadFile(spaIndex)
	if err != nil {
		log.Info("SPA index.html not found: ", err)
		Blank(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy",
		"default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self'; connect-src 'self'")
	w.Write(content)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
