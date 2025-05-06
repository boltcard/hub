package web

import (
	"card/db"
	"card/util"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
)

// {"enable":true,"card_name":"53a23d27e36d450f630e9a5f727dc239:5aa520a9fa5e9a016bde4125940bc005","tx_max":"1000","day_max":"10000","enable_pin":"false","pin_limit_sats":"100"}
// {"enable":"false","card_name":"53a23d27e36d450f630e9a5f727dc239:5aa520a9fa5e9a016bde4125940bc005","tx_max":"1000","day_max":"10000","enable_pin":true,"pin_limit_sats":"0"}
// {"enable":"false","card_name":"53a23d27e36d450f630e9a5f727dc239:5aa520a9fa5e9a016bde4125940bc005","tx_max":"1000","day_max":"10000","enable_pin":"true","pin_limit_sats":"100","card_pin_number":"1256"}

type UpdateCardWithPinRequest struct {
	Enable        bool   `json:"enable"`
	CardName      string `json:"card_name"`
	TxMax         string `json:"tx_max"`
	DayMax        string `json:"day_max"`
	EnablePin     bool   `json:"enable_pin"`
	PinLimitSats  string `json:"pin_limit_sats"`
	CardPinNumber string `json:"card_pin_number"`
}

type UpdateCardWithPinResponse struct {
	Status string `json:"status"`
}

func (app *App) CreateHandler_WalletApi_UpdateCardWithPin() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		log.Info("updateCardWithPin request received")

		// get access_token

		authToken := r.Header.Get("Authorization")
		splitToken := strings.Split(authToken, "Bearer ")
		accessToken := splitToken[1]

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		// get card_id from access_token

		card_id := db.Db_get_card_id_from_access_token(app.db_conn, accessToken)

		if card_id == 0 {
			sendError(w, "Bad auth", 1, "no card found for access token")
			return
		}

		// get details from request body
		// update card record

		body, err := io.ReadAll(r.Body)

		if err != nil {
			sendError(w, "Bad param", 8, "bad parameter passed in - read")
			return
		}

		bodyString := string(body)
		bodyString = strings.ReplaceAll(bodyString, `"true"`, "true")
		bodyString = strings.ReplaceAll(bodyString, `"false"`, "false")

		decoder := json.NewDecoder(strings.NewReader(bodyString))

		var reqObj UpdateCardWithPinRequest
		err = decoder.Decode(&reqObj)

		if err != nil {
			sendError(w, "Bad param", 8, "bad parameter passed in - decode")
			return
		}

		lnurlw_enable := "N"
		if reqObj.Enable {
			lnurlw_enable = "Y"
		}

		pin_enable := "N"
		if reqObj.EnablePin {
			pin_enable = "Y"
		}

		tx_limit_sats, err := strconv.Atoi(reqObj.TxMax)
		if err != nil {
			sendError(w, "Bad param", 8, "bad parameter passed in - tx_limit_sats")
			return
		}

		day_limit_sats, err := strconv.Atoi(reqObj.DayMax)
		if err != nil {
			sendError(w, "Bad param", 8, "bad parameter passed in - day_limit_sats")
			return
		}

		pin_limit_sats, err := strconv.Atoi(reqObj.PinLimitSats)
		if err != nil {
			sendError(w, "Bad param", 8, "bad parameter passed in - pin_limit_sats")
			return
		}

		if reqObj.CardPinNumber == "" {
			db.Db_update_card_without_pin(app.db_conn, card_id, tx_limit_sats, day_limit_sats, pin_enable, pin_limit_sats, lnurlw_enable)
		} else {
			db.Db_update_card_with_pin(app.db_conn, card_id, tx_limit_sats, day_limit_sats, pin_enable, reqObj.CardPinNumber, pin_limit_sats, lnurlw_enable)
		}

		var resObj UpdateCardWithPinResponse

		resObj.Status = "OK"

		resJson, err := json.Marshal(resObj)
		util.CheckAndPanic(err)

		w.Write(resJson)
	}
}
