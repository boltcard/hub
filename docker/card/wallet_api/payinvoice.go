package wallet_api

import (
	"card/db"
	"card/util"

	"card/phoenix"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	decodepay "github.com/nbd-wtf/ln-decodepay"
	log "github.com/sirupsen/logrus"
)

type PayInvoiceRequest struct {
	Invoice string `json:"invoice"`
	Amount  int    `json:"amount"`
}

type PayInvoiceResponse struct {
	Status string `json:"status"`
}

func PayInvoice(w http.ResponseWriter, r *http.Request) {

	log.Info("payinvoice request received")

	// set response header

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// get access_token

	authToken := r.Header.Get("Authorization")
	splitToken := strings.Split(authToken, "Bearer ")
	accessToken := splitToken[1]

	// get card_id from access_token

	card_id := db.Db_get_card_id_from_access_token(accessToken)
	log.Info("card_id ", card_id)

	if card_id == 0 {
		sendError(w, "Bad auth", 1, "no card found for access token")
		return
	}

	// get details from request body

	decoder := json.NewDecoder(r.Body)
	var reqObj PayInvoiceRequest
	err := decoder.Decode(&reqObj)
	if err != nil {
		sendError(w, "Error", 999, "request parameters invalid")
		return
	}

	bolt11, _ := decodepay.Decodepay(reqObj.Invoice)

	log.Info("decoded invoice ", bolt11)

	invAmtSat := int(bolt11.MSatoshi / 1000)

	log.Info("invAmtSat ", invAmtSat)
	log.Info("reqObj.Amount ", reqObj.Amount)

	if invAmtSat != 0 && reqObj.Amount != invAmtSat {
		sendError(w, "Error", 999, "invoice amounts don't match")
		return
	}

	actualAmtSat := util.Max(invAmtSat, reqObj.Amount)

	// check if there is sufficient balance

	total_paid_receipts := db.Db_get_total_paid_receipts(card_id)
	total_paid_payments := db.Db_get_total_payments(card_id)
	total_card_balance := total_paid_receipts - total_paid_payments

	if actualAmtSat > total_card_balance {
		sendError(w, "Error", 999, "invoice amount too large")
		return
	}

	// note that the order matters here -
	//  the balance must be reduced in the database before the payment is made
	// insert card_payment record

	db.Db_add_card_payment(card_id, reqObj.Amount, reqObj.Invoice)

	// make payment

	var payInvoiceRequest phoenix.PayInvoiceRequest

	payInvoiceRequest.Invoice = reqObj.Invoice
	payInvoiceRequest.AmountSat = strconv.Itoa(reqObj.Amount)

	payInvoiceResponse, err := phoenix.PayInvoice(payInvoiceRequest)
	util.Check(err)

	log.Info("payInvoiceResponse ", payInvoiceResponse)

	// create the response

	var resObj PayInvoiceResponse

	resObj.Status = "OK"

	resJson, err := json.Marshal(resObj)
	util.Check(err)

	log.Info("resJson ", string(resJson))

	w.Write(resJson)
}
