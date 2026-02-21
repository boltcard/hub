package web

import (
	"card/db"

	"card/phoenix"
	"card/util"
	"encoding/json"
	"net/http"
	"strconv"

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

		var resObj PayInvoiceResponse
		resObj.Status = "OK"

		writeJSON(w, resObj)
	}
}
