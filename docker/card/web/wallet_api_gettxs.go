package web

import (
	"card/db"

	"net/http"
	"strconv"

	_ "github.com/mattn/go-sqlite3"
	log "github.com/sirupsen/logrus"
)

type Transaction struct {
	PaymentPreimage string `json:"payment_preimage,omitempty"`
	PaymentHash     struct {
		Type string `json:"type"`
		Data []int  `json:"data"`
	} `json:"payment_hash,omitempty"`
	Type      string `json:"type"`
	Fee       int    `json:"fee"`
	Value     int    `json:"value"`
	Timestamp string `json:"timestamp"`
	Memo      string `json:"memo"`
}

type Transactions []Transaction

// returns invoices paid
func (app *App) CreateHandler_GetTxs() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		log.Info("getTxs request received")

		card_id, ok := app.getAuthenticatedCardID(w, r)
		if !ok {
			return
		}

		// query database card payments for card

		cardPayments := db.Db_select_card_payments(app.db_conn, card_id)

		var resObj Transactions
		resObj = make([]Transaction, 0)
		var tx Transaction

		for _, cardPayment := range cardPayments {
			tx.PaymentHash.Type = "Buffer"
			tx.Type = "paid_invoice"
			tx.Fee = cardPayment.FeeSats
			tx.Value = cardPayment.AmountSats
			tx.Timestamp = strconv.Itoa(cardPayment.Timestamp)
			tx.Memo = "" // TODO: add this

			resObj = append(resObj, tx)
		}

		writeJSON(w, resObj)
	}
}
