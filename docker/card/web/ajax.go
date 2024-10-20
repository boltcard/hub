package web

import (
	"encoding/json"
	"net/http"

	log "github.com/sirupsen/logrus"
)

type BalanceResponse struct {
	AvailableBalance int    `json:"AvailableBalance"`
	Error            string `json:"error,omitempty"`
}

func BalanceAjaxPage(w http.ResponseWriter, r *http.Request) {
	log.Info("balanceAjax request received")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	var resObj BalanceResponse
	resObj.AvailableBalance = 1234

	resJson, err := json.Marshal(resObj)
	if err != nil {
		panic(err)
	}

	log.Info("resJson string ", string(resJson))

	w.Write(resJson)
}
