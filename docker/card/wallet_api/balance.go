package wallet_api

import (
	"card/db"
	"card/util"

	"encoding/json"
	"net/http"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	log "github.com/sirupsen/logrus"
)

type BalanceResponse struct {
	BTC struct {
		AvailableBalance int `json:"AvailableBalance"`
	} `json:"BTC"`
	Error string `json:"error,omitempty"`
}

func Balance(w http.ResponseWriter, r *http.Request) {
	log.Info("balance request received")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// get access_token

	authToken := r.Header.Get("Authorization")
	splitToken := strings.Split(authToken, "Bearer ")
	accessToken := splitToken[1]

	// get card_id from access_token

	card_id := db.Db_get_card_id_from_access_token(accessToken)

	if card_id == 0 {
		sendError(w, "Bad auth", 1, "no card found for access token")
		return
	}

	// get the totals of paid transactions from the database

	//select IFNULL(SUM(amount_sats),0) from card_receipts where paid_flag='Y';

	//TODO: find out how we can tell if a payment will not get paid (hard fail)
	//select IFNULL(SUM(amount_sats),0) from card_payments;

	total_paid_receipts := db.Db_get_total_paid_receipts(card_id)
	total_paid_payments := db.Db_get_total_payments(card_id)
	total_card_balance := total_paid_receipts - total_paid_payments

	log.Info("total_card_balance = ", total_card_balance)

	var resObj BalanceResponse
	resObj.BTC.AvailableBalance = total_card_balance

	resJson, err := json.Marshal(resObj)
	util.Check(err)

	//	log.Info("resJson string ", string(resJson))

	w.Write(resJson)
}
