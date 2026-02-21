package web

import (
	"card/db"
	"strconv"

	"net/http"

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

func (app *App) CreateHandler_WalletApi_GetCard() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		log.Info("getCard request received")

		card_id, ok := app.getAuthenticatedCardID(w, r)
		if !ok {
			return
		}

		c, err := db.Db_get_card(app.db_conn, card_id)
		if err != nil {
			log.Error("db get card error: ", err)
			sendError(w, "Error", 999, "failed to get card")
			return
		}

		var resObj CardResponse

		resObj.Status = "OK"
		resObj.UID = c.Uid
		resObj.LnurlwEnable = c.Lnurlw_enable
		resObj.TxLimitSats = strconv.Itoa(c.Tx_limit_sats)
		resObj.DayLimitSats = strconv.Itoa(c.Day_limit_sats)
		resObj.PinEnable = c.Pin_enable
		resObj.PinLimitSats = strconv.Itoa(c.Pin_limit_sats)

		writeJSON(w, resObj)
	}
}
