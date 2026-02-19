package web

import (
	"card/db"

	"encoding/json"
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

		// set response header

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		// get access_token

		accessToken, ok := getBearerToken(w, r)
		if !ok {
			return
		}

		// get card_id from access_token

		card_id := db.Db_get_card_id_from_access_token(app.db_conn, accessToken)

		if card_id == 0 {
			sendError(w, "Bad auth", 1, "no card found for access token")
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

		resJson, err := json.Marshal(resObj)
		if err != nil {
			log.Error("json marshal error: ", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		w.Write(resJson)
	}
}
