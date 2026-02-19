package web

import (
	"card/db"
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

type AjaxBalanceResponse struct {
	AvailableBalance int    `json:"AvailableBalance"`
	Txs              []Tx   `json:"txs"`
	Error            string `json:"error,omitempty"`
}

func (app *App) CreateHandler_BalanceAjaxPage() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

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
		p, c, err := Get_p_c(u)
		if err != nil {
			return
		}

		cardMatch, cardId, cardCounter := Find_card(app.db_conn, p, c)

		if !cardMatch {
			return
		}

		// check counter is incremented
		cardLastCounter := db.Db_get_card_counter(app.db_conn, cardId)
		if cardCounter <= cardLastCounter {
			return
		}

		// store new counter value
		db.Db_set_card_counter(app.db_conn, cardId, cardCounter)

		log.Info("card_id = " + strconv.Itoa(cardId))

		// build response
		var resObj AjaxBalanceResponse

		// check the card balance
		total_card_balance := db.Db_get_card_balance(app.db_conn, cardId)
		resObj.AvailableBalance = total_card_balance

		// get card transactions
		cardTxs := db.Db_select_card_txs(app.db_conn, cardId)
		for _, cardTx := range cardTxs {
			// log.Info("cardId=" + strconv.Itoa(cardId) +
			// 	", AmountSats=" + strconv.Itoa(cardTx.AmountSats) +
			// 	", FeeSats=" + strconv.Itoa(cardTx.FeeSats) +
			// 	", Timestamp=" + strconv.Itoa(cardTx.Timestamp))
			var cardTxAppend Tx
			cardTxAppend.AmountSats = cardTx.AmountSats
			cardTxAppend.FeeSats = cardTx.FeeSats
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
}
