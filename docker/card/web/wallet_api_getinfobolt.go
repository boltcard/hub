package web

import (
	"net/http"

	log "github.com/sirupsen/logrus"
)

func (app *App) CreateHandler_GetInfoBolt() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		log.Info("getInfoBolt request received")
		w.Write([]byte(""))
	}
}
