package web

import (
	"card/db"

	"encoding/json"
	"net/http"

	_ "github.com/mattn/go-sqlite3"
	log "github.com/sirupsen/logrus"
)

type WipeCardResponse struct {
	Status  string `json:"status"`
	Action  string `json:"action"`
	Id      int    `json:"id"`
	Key0    string `json:"key0"`
	Key1    string `json:"key1"`
	Key2    string `json:"key2"`
	Key3    string `json:"key3"`
	Key4    string `json:"key4"`
	Uid     string `json:"uid"`
	Version int    `json:"version"`
}

func (app *App) CreateHandler_WalletApi_WipeCard() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		log.Info("wipeCard request received")

		// set response header

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

		// get card keys and deactivate card

		cardKeys := db.Db_wipe_card(app.db_conn, card_id)

		var resObj WipeCardResponse

		resObj.Status = "OK"
		resObj.Version = 1
		resObj.Action = "wipe"
		resObj.Id = 12
		resObj.Key0 = cardKeys.Key0
		resObj.Key1 = cardKeys.Key1
		resObj.Key2 = cardKeys.Key2
		resObj.Key3 = cardKeys.Key3
		resObj.Key4 = cardKeys.Key4
		resObj.Uid = "12345678"

		resJson, err := json.Marshal(resObj)
		if err != nil {
			log.Error("json marshal error: ", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		log.Info("resJson ", string(resJson))

		w.Write(resJson)
	}
}
