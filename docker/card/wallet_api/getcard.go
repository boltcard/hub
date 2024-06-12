package wallet_api

import (
	"card/db"
	"card/util"
	"strconv"

	"encoding/json"
	"net/http"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	log "github.com/sirupsen/logrus"
)

type CardResponse struct {
	Status       string `json:"status"`
	UID          string `json:"uid"`
	LnurlwEnable string `json:"lnurlw_enable"`
	TxLimitSats  string `json:"tx_limit_sats"`
	DayLimitSats string `json:"day_limit_sats"`
	PinEnable    string `json:"pin_enable"`
	PinLimitSats string `json:"pin_limit_sats"`
}

func GetCard(w http.ResponseWriter, r *http.Request) {
	log.Info("getCard request received")

	// get access_token

	authToken := r.Header.Get("Authorization")
	splitToken := strings.Split(authToken, "Bearer ")
	accessToken := splitToken[1]

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// get card_id from access_token

	card_id := db.Db_get_card_id_from_access_token(accessToken)

	if card_id == 0 {
		sendError(w, "Bad auth", 1, "no card found for access token")
		return
	}

	c, err := db.Db_get_card(card_id)
	util.Check(err)

	var resObj CardResponse

	resObj.Status = "OK"
	resObj.UID = c.Uid
	resObj.LnurlwEnable = c.Lnurlw_enable
	resObj.TxLimitSats = strconv.Itoa(c.Tx_limit_sats)
	resObj.DayLimitSats = strconv.Itoa(c.Day_limit_sats)
	resObj.PinEnable = c.Pin_enable
	resObj.PinLimitSats = strconv.Itoa(c.Pin_limit_sats)

	resJson, err := json.Marshal(resObj)
	util.Check(err)

	log.Info("resJson ", string(resJson))

	w.Write(resJson)
}
