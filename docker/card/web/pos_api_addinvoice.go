package web

import (
	"card/phoenix"
	"encoding/json"
	"net/http"
	"strconv"

	log "github.com/sirupsen/logrus"
)

type PosApiAddInvoiceRequest struct {
	Amt  string
	Memo string
}

type RHash struct {
	Type string `json:"type"`
	Data []int  `json:"data"`
}

type PosApiAddInvoiceResponse struct {
	PayReq         string `json:"pay_req"`
	PaymentRequest string `json:"payment_request"`
	AddIndex       string `json:"add_index"`
	RHash          RHash  `json:"r_hash"`
	Hash           string `json:"hash"`
}

func (app *App) CreateHandler_PosApi_AddInvoice() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		log.Info("pos_api AddInvoice request received")

		// reqToken := r.Header.Get("Authorization")
		// log.Info("auth token : ", reqToken)

		// get details from request body

		decoder := json.NewDecoder(r.Body)
		var reqObj PosApiAddInvoiceRequest
		err := decoder.Decode(&reqObj)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		amountSats, err := strconv.Atoi(reqObj.Amt)
		if err != nil {
			http.Error(w, "invalid amount", http.StatusBadRequest)
			return
		}

		if amountSats <= 0 {
			http.Error(w, "amount must be positive", http.StatusBadRequest)
			return
		}

		log.Info("AddInvoice : ", reqObj)

		// create invoice on Phoenix server
		invoiceRequest := phoenix.CreateInvoiceRequest{
			Description: reqObj.Memo,
			AmountSat:   reqObj.Amt,
			ExternalId:  "ext_id",
		}
		invoice, err := phoenix.CreateInvoice(invoiceRequest)
		if err != nil {
			log.Error("phoenix error: ", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		log.Info("invoice : ", invoice)

		// set up response
		var resp AddInvoiceResponse

		resp.PaymentRequest = invoice.Serialized
		resp.PayReq = resp.PaymentRequest
		// dummy data to pass BoltCardPos check
		resp.AddIndex = "500"
		resp.RHash.Type = "Buffer"
		resp.RHash.Data = []int{1, 2, 3}
		resp.Hash = invoice.PaymentHash

		respJson, err := json.Marshal(resp)
		if err != nil {
			log.Error("json marshal error: ", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		log.Info(string(respJson))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(respJson)
	}
}
