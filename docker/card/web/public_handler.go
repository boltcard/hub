package web

import (
	"net/http"

	log "github.com/sirupsen/logrus"
)

// all requests have a path prefix of /public/
// and can be returned without an authenticated session
func (app *App) CreateHandler_Public() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		request := r.RequestURI
		log.Info("CreateHandler_Public handler with request uri : " + request)

		// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Cache-Control
		w.Header().Add("Cache-Control", "no-cache")

		RenderStaticContent(w, request)
	}
}
