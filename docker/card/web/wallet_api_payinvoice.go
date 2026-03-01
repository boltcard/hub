package web

import (
	"card/db"
	"card/phoenix"
	"encoding/json"
	"math"
	"net/http"
	"strconv"
	"time"

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

func (app *App) CreateHandler_WalletApi_PayInvoice() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		log.Info("payinvoice request received")

		card_id, ok := app.getAuthenticatedCardID(w, r)
		if !ok {
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

		bolt11, err := decodepay.Decodepay(reqObj.Invoice)
		if err != nil {
			sendError(w, "Error", 999, "invalid invoice")
			return
		}

		if bolt11.MSatoshi < 0 || bolt11.MSatoshi > math.MaxInt64-999 {
			sendError(w, "Error", 999, "invalid invoice amount")
			return
		}
		invAmtSat := int(bolt11.MSatoshi / 1000)

		log.Info("invAmtSat ", invAmtSat)
		log.Info("reqObj.Amount ", reqObj.Amount)

		if invAmtSat != 0 && reqObj.Amount != invAmtSat {
			sendError(w, "Error", 999, "invoice amounts don't match")
			return
		}

		actualAmtSat := max(invAmtSat, reqObj.Amount)

		// check for duplicate payment
		if db.Db_get_paid_payment_exists(app.db_conn, reqObj.Invoice) {
			sendError(w, "Error", 999, "invoice already paid")
			return
		}

		// check if there is sufficient balance (atomic query)

		total_card_balance := db.Db_get_card_balance(app.db_conn, card_id)

		if actualAmtSat > total_card_balance {
			sendError(w, "Error", 999, "invoice amount too large")
			return
		}

		// note that the order matters here -
		//  the balance must be reduced in the database before the payment is made
		// insert card_payment record

		db.Db_add_card_payment(app.db_conn, card_id, reqObj.Amount, reqObj.Invoice)

		// make payment

		var payInvoiceRequest phoenix.SendLightningPaymentRequest

		payInvoiceRequest.Invoice = reqObj.Invoice
		payInvoiceRequest.AmountSat = strconv.Itoa(reqObj.Amount)

		payInvoiceResponse, payInvoiceResult, err := phoenix.SendLightningPayment(payInvoiceRequest)

		if err != nil {
			log.Error("Phoenix error response : ", err)
		}

		log.Info("payInvoiceResult : ", payInvoiceResult)
		log.Info("payInvoiceResponse : ", payInvoiceResponse)

		// create the response

		// broadcast to websocket clients
		app.broadcastPaymentSent(reqObj.Amount, bolt11.PaymentHash, time.Now().Unix())

		var resObj PayInvoiceResponse
		resObj.Status = "OK"

		writeJSON(w, resObj)
	}
}
