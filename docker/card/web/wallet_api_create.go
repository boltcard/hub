package web

import (
	"card/db"
	"card/util"

	"encoding/json"
	"net/http"

	_ "github.com/mattn/go-sqlite3"
	log "github.com/sirupsen/logrus"
)

type CreateResponse struct {
	Login    string `json:"login"`
	Password string `json:"password"`
	Error    string `json:"error"`
}

func (app *App) CreateHandler_Create() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		log.Info("create request received")

		decoder := json.NewDecoder(r.Body)
		t := struct {
			InviteSecret string `json:"invite_secret"`
		}{}

		err := decoder.Decode(&t)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// check the 'invite_secret' if configured
		db_invite_secret := db.Db_get_setting(app.db_conn, "invite_secret")
		req_invite_secret := t.InviteSecret

		if req_invite_secret != db_invite_secret {
			sendError(w, "Bad auth", 1, "incorrect invite_secret")
			return
		}

		// create a new card account in the database
		key0, key1, k2, key3, key4 := generateCardKeys()
		login := util.Random_hex()
		password := util.Random_hex()

		db.Db_insert_card(app.db_conn, key0, key1, k2, key3, key4, login, password)

		// return the login & password for the card account

		var resObj CreateResponse

		resObj.Login = login
		resObj.Password = password

		writeJSON(w, resObj)
	}
}
