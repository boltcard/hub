package web

import (
	"io"
	"net/http"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"
)

func Admin(w http.ResponseWriter, r *http.Request) {
	request := r.RequestURI

	// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Cache-Control
	w.Header().Add("Cache-Control", "no-cache")

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

	content, err := os.Open("/web-content" + request)

	if err != nil {
		log.Info(err.Error())
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
