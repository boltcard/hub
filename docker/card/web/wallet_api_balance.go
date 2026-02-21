package web

import (
	"card/db"

	"net/http"

	_ "github.com/mattn/go-sqlite3"
	log "github.com/sirupsen/logrus"
)

type BalanceResponse struct {
	BTC struct {
		AvailableBalance int `json:"AvailableBalance"`
	} `json:"BTC"`
	Error string `json:"error,omitempty"`
}

func (app *App) CreateHandler_Balance() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		log.Info("balance request received")

		card_id, ok := app.getAuthenticatedCardID(w, r)
		if !ok {
			return
		}

		// get the card balance atomically

		total_card_balance := db.Db_get_card_balance(app.db_conn, card_id)

		log.Info("total_card_balance = ", total_card_balance)

		var resObj BalanceResponse
		resObj.BTC.AvailableBalance = total_card_balance

		writeJSON(w, resObj)
	}
}
