package wallet_api

import (
	"card/util"
	"encoding/json"
	"net/http"
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
	util.Check(err)
	w.Write(resJson)
}
