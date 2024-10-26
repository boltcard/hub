package web

import (
	"card/db"
	"card/lnurlw"
	"encoding/json"
	"net/http"
	"net/url"

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

	log.Info("card read = " + card_str)

	u, err := url.Parse(card_str)
	if err != nil {
		return
	}

	// TODO: check card domain and advise if incorrect

	// check card p & c values
	p, c, err := lnurlw.Get_p_c(u)
	if err != nil {
		return
	}

	cardMatch, cardId, cardCounter := lnurlw.Find_card(p, c)

	// check counter is incremented
	cardLastCounter := db.Db_get_card_counter(cardId)
	if cardCounter <= cardLastCounter {
		return
	}

	// store new counter value
	db.Db_set_card_counter(cardId, cardCounter)

	var resObj BalanceResponse
	resObj.AvailableBalance = -1

	if cardMatch {
		resObj.AvailableBalance = cardId
	}

	resJson, err := json.Marshal(resObj)
	if err != nil {
		return
	}

	//log.Info("resJson string ", string(resJson))

	w.Write(resJson)
}
