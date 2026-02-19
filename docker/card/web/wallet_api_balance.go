package web

import (
	"card/db"

	"encoding/json"
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

		// get the card balance atomically

		total_card_balance := db.Db_get_card_balance(app.db_conn, card_id)

		log.Info("total_card_balance = ", total_card_balance)

		var resObj BalanceResponse
		resObj.BTC.AvailableBalance = total_card_balance

		resJson, err := json.Marshal(resObj)
		if err != nil {
			log.Error("json marshal error: ", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		w.Write(resJson)
	}
}
