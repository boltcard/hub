package web

import (
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
