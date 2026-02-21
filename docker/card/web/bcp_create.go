package web

import (
	"card/db"
	"card/util"
	"net/http"

	log "github.com/sirupsen/logrus"
)

type BcpResponse struct {
	ProtocolName    string `json:"protocol_name"`
	ProtocolVersion int    `json:"protocol_version"`
	CardName        string `json:"card_name"`
	LnurlwBase      string `json:"lnurlw_base"`
	UIDPrivacy      string `json:"uid_privacy"`
	K0              string `json:"k0"`
	K1              string `json:"k1"`
	K2              string `json:"k2"`
	K3              string `json:"k3"`
	K4              string `json:"k4"`
}

func (app *App) CreateHandler_CreateCard() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		param_a := r.URL.Query().Get("a")

		log.Info("CreateCard")

		if param_a == "" {
			w.Write([]byte(`{"status": "ERROR", "reason": "a value not found"}`))
			return
		}

		if param_a != db.Db_get_setting(app.db_conn, "new_card_code") {
			w.Write([]byte(`{"status": "ERROR", "reason": "a value not valid"}`))
			return
		}

		// create a new card in the database
		k0, k1, k2, k3, k4 := generateCardKeys()
		login := util.Random_hex()
		password := util.Random_hex()
		db.Db_insert_card(app.db_conn, k0, k1, k2, k3, k4, login, password)

		var resObj BcpResponse

		resObj.ProtocolName = "new_bolt_card_response"
		resObj.ProtocolVersion = 1
		resObj.CardName = "Spending_Card"
		resObj.LnurlwBase = "lnurlw://" + db.Db_get_setting(app.db_conn, "host_domain") + "/ln"
		resObj.UIDPrivacy = "Y"
		resObj.K0 = k0
		resObj.K1 = k1
		resObj.K2 = k2
		resObj.K3 = k3
		resObj.K4 = k4

		writeJSON(w, resObj)
	}
}
