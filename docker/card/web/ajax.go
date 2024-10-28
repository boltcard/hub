package web

import (
	"card/db"
	"card/lnurlw"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"

	log "github.com/sirupsen/logrus"
)

type Tx struct {
	AmountSats int `json:"AmountSats"`
	FeeSats    int `json:"FeeSats"`
	Timestamp  int `json:"Timestamp"`
}

type BalanceResponse struct {
	AvailableBalance int    `json:"AvailableBalance"`
	Txs              []Tx   `json:"txs"`
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

	if !cardMatch {
		return
	}

	// check counter is incremented
	cardLastCounter := db.Db_get_card_counter(cardId)
	if cardCounter <= cardLastCounter {
		return
	}

	// store new counter value
	db.Db_set_card_counter(cardId, cardCounter)

	log.Info("card_id = " + strconv.Itoa(cardId))

	// build response
	var resObj BalanceResponse

	// check the card balance
	total_paid_receipts := db.Db_get_total_paid_receipts(cardId)
	total_paid_payments := db.Db_get_total_paid_payments(cardId)
	total_card_balance := total_paid_receipts - total_paid_payments
	resObj.AvailableBalance = total_card_balance

	// get card transactions
	cardTxs := db.Db_select_card_txs(cardId)
	for _, cardTx := range cardTxs {
		// log.Info("cardId=" + strconv.Itoa(cardId) +
		// 	", AmountSats=" + strconv.Itoa(cardTx.AmountSats) +
		// 	", FeeSats=" + strconv.Itoa(cardTx.FeeSats) +
		// 	", Timestamp=" + strconv.Itoa(cardTx.Timestamp))
		var cardTxAppend Tx
		cardTxAppend.AmountSats = cardTx.AmountSats
		cardTxAppend.FeeSats = 0
		cardTxAppend.Timestamp = cardTx.Timestamp
		resObj.Txs = append(resObj.Txs, cardTxAppend)
	}

	resJson, err := json.Marshal(resObj)
	if err != nil {
		return
	}

	//log.Info("resJson string ", string(resJson))

	w.Write(resJson)
}
