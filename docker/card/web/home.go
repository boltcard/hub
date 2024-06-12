package web

import (
	"net/http"

	log "github.com/sirupsen/logrus"
)

func HomePage(w http.ResponseWriter, r *http.Request) {
	log.Info("homepage request received")

	renderTemplate(w, "home", nil)
}
