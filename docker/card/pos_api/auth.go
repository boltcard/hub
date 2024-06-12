package pos_api

import (
	"net/http"

	log "github.com/sirupsen/logrus"
)

type AuthRequest struct {
	Login    string
	Password string
}

func Auth(w http.ResponseWriter, r *http.Request) {
	log.Info("pos_api Auth request received")

	var a AuthRequest

	// err := decodeJSONBody(w, r, &a)
	// if err != nil {
	// 	var mr *malformedRequest
	// 	if errors.As(err, &mr) {
	// 		log.Error(mr.msg)
	// 	} else {
	// 		log.Error(err.Error())
	// 	}
	// 	return
	// }

	log.Info("Auth : ", a)

	//TODO: check authorisation login & password
	//TODO: generate and store in database the refresh_token and access_token with timeouts

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	jsonData := []byte(`{"refresh_token":"aa","access_token":"bb"}`)
	w.Write(jsonData)
}
