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

	params_p, ok := r.URL.Query()["card"]
	if !ok || len(params_p[0]) < 1 {
		log.Info("card value not found")
		//TODO: return error
	}

	card_str := params_p[0]

	log.Info("card_p = " + card_str)

	var resObj BalanceResponse
	resObj.AvailableBalance = 1234

	resJson, err := json.Marshal(resObj)
	if err != nil {
		panic(err)
	}

	//log.Info("resJson string ", string(resJson))

	w.Write(resJson)
}
