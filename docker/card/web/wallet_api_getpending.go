package web

import (
	"net/http"

	log "github.com/sirupsen/logrus"
)

func (app *App) CreateHandler_GetPending() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		log.Info("getPending request received")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		jsonData := []byte(`[]`) // array
		w.Write(jsonData)
	}
}
