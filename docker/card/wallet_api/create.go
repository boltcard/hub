package wallet_api

import (
	"card/db"
	"card/util"

	"net/http"

	_ "github.com/mattn/go-sqlite3"
	log "github.com/sirupsen/logrus"

	//	"card/phoenix"
	"encoding/json"
)

type CreateResponse struct {
	Login    string `json:"login"`
	Password string `json:"password"`
	Error    string `json:"error"`
}

func Create(w http.ResponseWriter, r *http.Request) {
	log.Info("create request received")

	decoder := json.NewDecoder(r.Body)
	//	decoder.DisallowUnknownFields()
	t := struct {
		InviteSecret string `json:"invite_secret"`
	}{}

	err := decoder.Decode(&t)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// check the 'invite_secret' if configured
	db_invite_secret := db.Db_get_setting("invite_secret")
	req_invite_secret := t.InviteSecret

	log.Info("db_invite_secret " + db_invite_secret)
	log.Info("req_invite_secret " + req_invite_secret)

	if req_invite_secret != db_invite_secret {
		sendError(w, "Bad auth", 1, "incorrect invite_secret")
		return
	}

	// create a new card account in the database
	key0 := util.Random_hex()
	key1 := util.Random_hex()
	k2 := util.Random_hex()
	key3 := util.Random_hex()
	key4 := util.Random_hex()
	login := util.Random_hex()
	password := util.Random_hex()

	db.Db_insert_card(key0, key1, k2, key3, key4, login, password)

	// return the login & password for the card account

	var resObj CreateResponse

	resObj.Login = login
	resObj.Password = password

	resJson, err := json.Marshal(resObj)
	util.Check(err)

	w.Write(resJson)
}
