package bcp

import (
	"card/db"
	"card/util"
	"encoding/json"
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

func CreateCard(w http.ResponseWriter, r *http.Request) {
	param_a := r.URL.Query().Get("a")

	log.Info("CreateCard a=" + param_a)

	if param_a == "" {
		w.Write([]byte(`{"status": "ERROR", "reason": "a value not found"}`))
		return
	}

	if param_a != db.Db_get_setting("new_card_code") {
		w.Write([]byte(`{"status": "ERROR", "reason": "a value not valid"}`))
		return
	}

	var resObj BcpResponse

	resObj.ProtocolName = "new_bolt_card_response"
	resObj.ProtocolName = "1"
	resObj.CardName = "Spending_Card"
	resObj.LnurlwBase = "lnurlw://" + db.Db_get_setting("host_domain") + "/ln"
	resObj.UIDPrivacy = "Y"
	resObj.K0 = "11111111111111111111111111111111"
	resObj.K1 = "22222222222222222222222222222222"
	resObj.K2 = "33333333333333333333333333333333"
	resObj.K3 = "44444444444444444444444444444444"
	resObj.K4 = "55555555555555555555555555555555"

	resJson, err := json.Marshal(resObj)
	util.Check(err)

	//log.Info("resJson ", string(resJson))

	w.Write(resJson)
}
