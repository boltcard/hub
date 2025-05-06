package web

import (
	"net/http"

	log "github.com/sirupsen/logrus"
)

func (app *App) CreateHandler_PosApi_GetInfo() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		log.Info("pos_api GetInfo request received")

		r.ParseForm()
		for key, values := range r.Form {
			log.Info(key, " = ", values)
		}

		w.Write([]byte(""))
	}
}
