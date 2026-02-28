package web

import (
	"card/db"
	"card/util"
	"encoding/json"
	"net/http"
	"strings"

	log "github.com/sirupsen/logrus"
)

type ErrorResponse struct {
	Error   string `json:"error"`
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func sendError(w http.ResponseWriter, error string, code int, message string) {
	var errorResponse ErrorResponse
	errorResponse.Error = error
	errorResponse.Code = code
	errorResponse.Message = message
	resJson, err := json.Marshal(errorResponse)
	if err != nil {
		log.Error("json marshal error: ", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(resJson)
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	resJson, err := json.Marshal(v)
	if err != nil {
		log.Error("json marshal error: ", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(resJson)
}

func getBearerToken(w http.ResponseWriter, r *http.Request) (string, bool) {
	authToken := r.Header.Get("Authorization")
	if !strings.HasPrefix(authToken, "Bearer ") {
		sendError(w, "Bad auth", 1, "missing or invalid Authorization header")
		return "", false
	}
	return authToken[7:], true
}

func generateCardKeys() (key0, key1, k2, key3, key4 string) {
	return util.Random_hex(), util.Random_hex(), util.Random_hex(), util.Random_hex(), util.Random_hex()
}

func (app *App) getAuthenticatedCardID(w http.ResponseWriter, r *http.Request) (int, bool) {
	accessToken, ok := getBearerToken(w, r)
	if !ok {
		return 0, false
	}

	card_id := db.Db_get_card_id_from_access_token(app.db_conn, accessToken)
	if card_id == 0 {
		sendError(w, "Bad auth", 1, "no card found for access token")
		return 0, false
	}

	return card_id, true
}
