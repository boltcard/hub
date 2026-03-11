package web

import (
	"card/db"
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
	CardId           int    `json:"CardId"`
	Note             string `json:"Note"`
	AvailableBalance int    `json:"AvailableBalance"`
	Txs              []Tx   `json:"txs"`
	Error            string `json:"error,omitempty"`
}

func (app *App) CreateHandler_BalanceAjaxPage() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		log.Info("balanceAjax request received")

		params_p, ok := r.URL.Query()["card"]
		if !ok || len(params_p[0]) < 1 {
			log.Info("card value not found")
			writeJSON(w, AjaxBalanceResponse{Error: "card value not found"})
			return
		}

		card_str := params_p[0]

		log.Info("card read = " + card_str)

		u, err := url.Parse(card_str)
		if err != nil {
			writeJSON(w, AjaxBalanceResponse{Error: "invalid card data"})
			return
		}

		// check card p & c values
		p, c, err := Get_p_c(u)
		if err != nil {
			writeJSON(w, AjaxBalanceResponse{Error: "card not recognised"})
			return
		}

		cardMatch, cardId, cardCounter := Find_card(app.db_read, p, c)

		if !cardMatch {
			writeJSON(w, AjaxBalanceResponse{Error: "card not found"})
			return
		}

		// check counter is incremented
		cardLastCounter := db.Db_get_card_counter(app.db_read, cardId)
		if cardCounter <= cardLastCounter {
			writeJSON(w, AjaxBalanceResponse{Error: "card already scanned, tap again"})
			return
		}

		// store new counter value
		db.Db_set_card_counter(app.db_write, cardId, cardCounter)

		log.Info("card_id = " + strconv.Itoa(cardId))

		// build response
		var resObj AjaxBalanceResponse

		resObj.CardId = cardId
		card, err := db.Db_get_card(app.db_read, cardId)
		if err == nil {
			resObj.Note = card.Note
		}

		// check the card balance
		total_card_balance := db.Db_get_card_balance(app.db_read, cardId)
		resObj.AvailableBalance = total_card_balance

		// get card transactions
		cardTxs := db.Db_select_card_txs(app.db_read, cardId)
		for _, cardTx := range cardTxs {
			var cardTxAppend Tx
			cardTxAppend.AmountSats = cardTx.AmountSats
			cardTxAppend.FeeSats = cardTx.FeeSats
			cardTxAppend.Timestamp = cardTx.Timestamp
			resObj.Txs = append(resObj.Txs, cardTxAppend)
		}

		writeJSON(w, resObj)
	}
}
