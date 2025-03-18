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

func GetCardKeys(w http.ResponseWriter, r *http.Request) {
	log.Info("getCardKeys request received")

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

	// create new random card keys in database
	key0 := util.Random_hex()
	key1 := util.Random_hex()
	k2 := util.Random_hex()
	key3 := util.Random_hex()
	key4 := util.Random_hex()

	// TODO: archive card keys

	db.Db_set_card_keys(card_id, key0, key1, k2, key3, key4)

	var resObj CardKeysResponse

	resObj.ProtocolName = "create_bolt_card_response"
	resObj.ProtocolVersion = 2
	resObj.CardName = "card"
	resObj.LnurlwBase = "lnurlw://" + db.Db_get_setting("host_domain") + "/ln"
	resObj.Key0 = key0
	resObj.Key1 = key1
	resObj.Key2 = k2
	resObj.Key3 = key3
	resObj.Key4 = key4
	resObj.UidPrivacy = "false"

	resJson, err := json.Marshal(resObj)
	util.CheckAndPanic(err)

	log.Info("resJson ", string(resJson))

	w.Write(resJson)
}
