package wallet_api

import (
	"card/db"
	"card/phoenix"
	"card/util"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	log "github.com/sirupsen/logrus"
)

type AddInvoiceRequest struct {
	Amt  string `json:"amt"`
	Memo string `json:"memo"`
}

type AddInvoiceResponse struct {
	PayReq         string `json:"pay_req"`
	PaymentRequest string `json:"payment_request"`
	AddIndex       string `json:"add_index"`
	RHash          struct {
		Type string `json:"type"`
		Data []int  `json:"data"`
	} `json:"r_hash"`
	Hash  string `json:"hash"`
	Error string `json:"error,omitempty"`
}

func AddInvoice(w http.ResponseWriter, r *http.Request) {

	log.Info("addinvoice request received")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// get access_token

	authToken := r.Header.Get("Authorization")
	splitToken := strings.Split(authToken, "Bearer ")
	accessToken := splitToken[1]

	log.Info("access_token ", accessToken)

	// get details from request body

	decoder := json.NewDecoder(r.Body)
	var reqObj AddInvoiceRequest
	err := decoder.Decode(&reqObj)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	amountSats, err := strconv.Atoi(reqObj.Amt)

	if amountSats <= 0 || err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// create an invoice

	var createInvoiceRequest phoenix.ReceiveLightningPaymentRequest

	createInvoiceRequest.Description = reqObj.Memo
	createInvoiceRequest.AmountSat = strconv.Itoa(amountSats)
	createInvoiceRequest.ExternalId = "" // could use a unique id here if needed

	createInvoiceResponse, err := phoenix.ReceiveLightningPayment(createInvoiceRequest)
	util.Check(err)

	log.Info("createInvoiceResponse ", createInvoiceResponse)

	// organise the invoice data

	var resObj AddInvoiceResponse

	rHashByteSlice, err := hex.DecodeString(createInvoiceResponse.PaymentHash)
	util.Check(err)

	rHashIntSlice := []int{}
	for _, rHashByte := range rHashByteSlice {
		rHashIntSlice = append(rHashIntSlice, int(rHashByte))
	}

	// save in our database

	// get card_id from access_token

	card_id := db.Db_get_card_id_from_access_token(accessToken)
	log.Info("card_id ", card_id)

	if card_id == 0 {
		sendError(w, "Bad auth", 1, "no card found for access token")
		return
	}

	// insert card_receipt record

	db.Db_add_card_receipt(card_id, createInvoiceResponse.Serialized, createInvoiceResponse.PaymentHash, amountSats)

	// create the invoice response

	resObj.PayReq = createInvoiceResponse.Serialized
	resObj.PaymentRequest = createInvoiceResponse.Serialized
	resObj.RHash.Type = "buffer"
	resObj.RHash.Data = rHashIntSlice
	resObj.Hash = createInvoiceResponse.PaymentHash

	resJson, err := json.Marshal(resObj)
	util.Check(err)

	log.Info("resJson ", string(resJson))

	w.Write(resJson)
}
