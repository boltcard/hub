package pos_api

import (
	"card/phoenix"
	"card/util"
	"encoding/json"
	"net/http"
	"strconv"

	log "github.com/sirupsen/logrus"
)

type AddInvoiceRequest struct {
	Amt  string
	Memo string
}

type RHash struct {
	Type string `json:"type"`
	Data []int  `json:"data"`
}

type AddInvoiceResponse struct {
	PayReq         string `json:"pay_req"`
	PaymentRequest string `json:"payment_request"`
	AddIndex       string `json:"add_index"`
	RHash          RHash  `json:"r_hash"`
	Hash           string `json:"hash"`
}

func AddInvoice(w http.ResponseWriter, r *http.Request) {
	log.Info("pos_api AddInvoice request received")

	// reqToken := r.Header.Get("Authorization")
	// log.Info("auth token : ", reqToken)

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

	log.Info("AddInvoice : ", reqObj)

	// create invoice on Phoenix server
	invoiceRequest := phoenix.CreateInvoiceRequest{
		Description: reqObj.Memo,
		AmountSat:   reqObj.Amt,
		ExternalId:  "ext_id",
	}
	invoice, err := phoenix.CreateInvoice(invoiceRequest)
	util.Check(err)

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
	util.Check(err)

	log.Info(string(respJson))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(respJson)
}
