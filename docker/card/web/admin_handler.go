package web

import (
	"net/http"
	"os"
	"strings"

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

		// Serve React SPA for all admin paths.
		// The SPA handles auth via /admin/api/auth/check.
		serveSpaIndex(w, r)
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
