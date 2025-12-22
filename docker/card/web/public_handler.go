package web

import (
	"net/http"
)

// all requests have a path prefix of /public/
// and can be returned without an authenticated session
func (app *App) CreateHandler_Public() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		request := r.RequestURI
		//log.Info("CreateHandler_Public handler with request uri : " + request)

		w.Header().Add("Cache-Control", "max-age=60, must-revalidate")

		RenderStaticContent(w, request)
	}
}
