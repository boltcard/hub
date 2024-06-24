package wallet_api

import (
	"net/http"

	log "github.com/sirupsen/logrus"
)

func GetInfoBolt(w http.ResponseWriter, r *http.Request) {
	log.Info("getInfoBolt request received")
	w.Write([]byte(""))
}

