package web

import (
	"card/db"

	"net/http"

	_ "github.com/mattn/go-sqlite3"
	log "github.com/sirupsen/logrus"
)

type CardKeysResponse struct {
	ProtocolName    string `json:"protocol_name"`
	ProtocolVersion int    `json:"protocol_version"`
	CardName        string `json:"card_name"`
	LnurlwBase      string `json:"lnurlw_base"`
	Key0            string `json:"key0"`
	Key1            string `json:"key1"`
	Key2            string `json:"k2"`
	Key3            string `json:"key3"`
	Key4            string `json:"key4"`
	UidPrivacy      string `json:"uid_privacy"`
}

func (app *App) CreateHandler_WalletApi_GetCardKeys() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		log.Info("getCardKeys request received")

		card_id, ok := app.getAuthenticatedCardID(w, r)
		if !ok {
			return
		}

		// create new random card keys in database
		key0, key1, k2, key3, key4 := generateCardKeys()

		// TODO: archive card keys

		db.Db_set_card_keys(app.db_conn, card_id, key0, key1, k2, key3, key4)

		var resObj CardKeysResponse

		resObj.ProtocolName = "create_bolt_card_response"
		resObj.ProtocolVersion = 2
		resObj.CardName = "card"
		resObj.LnurlwBase = "lnurlw://" + db.Db_get_setting(app.db_conn, "host_domain") + "/ln"
		resObj.Key0 = key0
		resObj.Key1 = key1
		resObj.Key2 = k2
		resObj.Key3 = key3
		resObj.Key4 = key4
		resObj.UidPrivacy = "false"

		log.Info("getCardKeys response prepared")

		writeJSON(w, resObj)
	}
}
