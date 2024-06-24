package wallet_api

import (
	"net/http"

	log "github.com/sirupsen/logrus"
)

func GetPending(w http.ResponseWriter, r *http.Request) {
	log.Info("getPending request received")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	jsonData := []byte(`[]`) // array
	w.Write(jsonData)
}
