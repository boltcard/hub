package web

import (
	"card/db"
	"card/util"

	"encoding/json"
	"net/http"

	log "github.com/sirupsen/logrus"
)

type AuthRequest struct {
	RefreshToken string `json:"refresh_token"`
	Login        string `json:"login"`
	Password     string `json:"password"`
}

type AuthResponse struct {
	RefreshToken string `json:"refresh_token"`
	AccessToken  string `json:"access_token"`
}

func (app *App) CreateHandler_Auth() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		log.Info("auth request received")

		authType := r.URL.Query().Get("type")

		// handle refresh_token (issue a new access_token)
		if authType == "refresh_token" {
			log.Info("auth using refresh_token")

			// get refresh_token from body
			decoder := json.NewDecoder(r.Body)
			var t AuthRequest
			err := decoder.Decode(&t)
			if err != nil {
				sendError(w, "Bad auth", 1, "bad format for refresh_token")
				return
			}

			// create new tokens
			RefreshToken := util.Random_hex()
			AccessToken := util.Random_hex()

			// update tokens
			success := db.Db_update_tokens(app.db_conn, t.RefreshToken, RefreshToken, AccessToken)

			if !success {
				sendError(w, "Bad auth", 1, "invalid refresh_token")
				return
			}

			// return access_token
			var resObj AuthResponse
			resObj.RefreshToken = RefreshToken
			resObj.AccessToken = AccessToken

			writeJSON(w, resObj)
			return
		}

		// handle login & password (issue a new refresh_token and access_token)
		if authType == "auth" {
			log.Info("auth using login & password")

			// get login & password from body
			decoder := json.NewDecoder(r.Body)
			var reqObj AuthRequest
			err := decoder.Decode(&reqObj)
			if err != nil {
				sendError(w, "Bad auth", 1, "bad format for login and password")
				return
			}

			log.Info("AuthRequest received for login")

			// create new refresh_token & access_token
			AccessToken := util.Random_hex()
			RefreshToken := util.Random_hex()

			// store tokens in database if there is a matching Login & Password
			err = db.Db_set_tokens(app.db_conn, reqObj.Login, reqObj.Password, AccessToken, RefreshToken)
			if err != nil {
				sendError(w, "Bad auth", 1, "invalid login and password")
				return
			}

			// return refresh_token & access_token
			var resObj AuthResponse
			resObj.RefreshToken = RefreshToken
			resObj.AccessToken = AccessToken

			log.Info("AuthResponse sent")

			writeJSON(w, resObj)
			return
		}

		sendError(w, "Bad auth", 1, "auth parameters not valid")
	}
}
