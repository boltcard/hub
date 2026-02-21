package web

import (
	"card/db"
	"card/phoenix"
	"card/util"
	"encoding/json"
	"net/http"
	"strconv"

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

func (app *App) CreateHandler_AddInvoice() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		log.Info("addinvoice request received")

		// validate auth before doing anything else
		card_id, ok := app.getAuthenticatedCardID(w, r)
		if !ok {
			return
		}

		// get details from request body

		decoder := json.NewDecoder(r.Body)
		var reqObj AddInvoiceRequest
		err := decoder.Decode(&reqObj)
		if err != nil {
			sendError(w, "Error", 8, "invalid request body")
			return
		}

		amountSats, err := strconv.Atoi(reqObj.Amt)
		if err != nil {
			sendError(w, "Error", 8, "invalid amount")
			return
		}

		if amountSats <= 0 {
			sendError(w, "Error", 8, "amount must be positive")
			return
		}

		// create an invoice

		var createInvoiceRequest phoenix.CreateInvoiceRequest

		createInvoiceRequest.Description = reqObj.Memo
		createInvoiceRequest.AmountSat = strconv.Itoa(amountSats)
		createInvoiceRequest.ExternalId = "" // could use a unique id here if needed

		createInvoiceResponse, err := phoenix.CreateInvoice(createInvoiceRequest)
		if err != nil {
			log.Error("phoenix CreateInvoice error: ", err)
			sendError(w, "Error", 999, "failed to create invoice")
			return
		}

		log.Info("createInvoiceResponse ", createInvoiceResponse)

		// organise the invoice data

		var resObj AddInvoiceResponse

		rHashIntSlice := util.ConvertPaymentHash(createInvoiceResponse.PaymentHash)

		// insert card_receipt record

		db.Db_add_card_receipt(app.db_conn, card_id,
			createInvoiceResponse.Serialized, createInvoiceResponse.PaymentHash, amountSats)

		// create the invoice response

		resObj.PayReq = createInvoiceResponse.Serialized
		resObj.PaymentRequest = createInvoiceResponse.Serialized
		resObj.RHash.Type = "buffer"
		resObj.RHash.Data = rHashIntSlice
		resObj.Hash = createInvoiceResponse.PaymentHash

		writeJSON(w, resObj)
	}
}
