package web

import (
	"card/db"

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

		card_id, ok := app.getAuthenticatedCardID(w, r)
		if !ok {
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

		writeJSON(w, resObj)
	}
}
