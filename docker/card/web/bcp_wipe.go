package web

import (
	"encoding/json"
	"net/http"

	log "github.com/sirupsen/logrus"
)

type BcpWipeCardRequest struct {
	Uid    string `json:"UID,omitempty"`
	Lnurlw string `json:"LNURLW,omitempty"`
}

func (app *App) CreateHandler_WipeCard() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		log.Info("BcpWipeCard request received")

		// get details from request body

		decoder := json.NewDecoder(r.Body)
		var reqObj BcpWipeCardRequest
		err := decoder.Decode(&reqObj)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		log.Infof("UID: %s", reqObj.Uid)
		log.Infof("LNURLW: %s", reqObj.Lnurlw)
	}
}
