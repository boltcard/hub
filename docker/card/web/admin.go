package web

import (
	"card/db"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

func renderContent(w http.ResponseWriter, r *http.Request) {
	request := r.RequestURI

	// default to index.html
	if strings.HasSuffix(request, "/") {
		request = request + "index.html"
	}

	// only log page requests
	if strings.HasSuffix(request, ".html") {
		log.Info("page : ", request)
		template_path := strings.Replace(request, "/admin/", "/dist/pages/admin/", 1)
		w.Header().Add("Content-Type", "text/html")
		renderTemplate(w, template_path, nil)
		return
	}

	// everything except .html
	content, err := os.Open("/web-content" + request)

	if err != nil {
		log.Info(err.Error())
		Blank(w, r)
		return
	}

	defer content.Close()

	switch {
	case strings.HasSuffix(request, ".js"):
		w.Header().Add("Content-Type", "application/json")
	case strings.HasSuffix(request, ".css"):
		w.Header().Add("Content-Type", "text/css")
	case strings.HasSuffix(request, ".png"):
		w.Header().Add("Content-Type", "image/png")
	case strings.HasSuffix(request, ".jpg"):
		w.Header().Add("Content-Type", "image/jpeg")
	default:
		log.Info("suffix not recognised : ", request)
		return
	}

	io.Copy(w, content)
}

func Admin(w http.ResponseWriter, r *http.Request) {
	request := r.RequestURI

	// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Cache-Control
	w.Header().Add("Cache-Control", "no-cache")

	// items to return without an authenticated session
	if strings.HasSuffix(request, ".js") || strings.HasSuffix(request, ".css") ||
		strings.HasSuffix(request, ".png") || strings.HasSuffix(request, ".jpg") {
		renderContent(w, r)
		return
	}

	// detect if an admin password has been set
	if db.Db_get_setting("admin_password_hash") == "" {
		if request == "/admin/register/" {
			http.SetCookie(w, &http.Cookie{
				Name:    "session_token",
				Value:   "",
				Expires: time.Now(),
			})
			renderContent(w, r)
			return
		} else {
			// https://freshman.tech/snippets/go/http-redirect/
			//redirect to "register" page
			http.Redirect(w, r, "/admin/register/", http.StatusSeeOther)
			return
		}
	}

	if request == "/admin/login/" {
		http.SetCookie(w, &http.Cookie{
			Name:    "session_token",
			Value:   "",
			Expires: time.Now(),
		})
		renderContent(w, r)
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
		log.Info("session_token error : " + err.Error())
		Blank(w, r)
		return
	}

	// validate the session cookie
	sessionToken := c.Value
	adminSessionToken := db.Db_get_setting("session_token")

	if sessionToken != adminSessionToken {
		http.SetCookie(w, &http.Cookie{
			Name:    "session_token",
			Value:   "",
			Expires: time.Now(),
		})
		//redirect to "login" page
		http.Redirect(w, r, "/admin/login/", http.StatusSeeOther)
		return
	}

	renderContent(w, r)
}
