package bcp

import (
	"net/http"

	log "github.com/sirupsen/logrus"
)

func CreateCard(w http.ResponseWriter, r *http.Request) {
	param_a := r.URL.Query().Get("a")

	if param_a == "" {
		w.Write([]byte(`{"status": "ERROR", "reason": "a value not found"}`))
		return
	}

	log.Info("CreateCard a=" + param_a)
}
