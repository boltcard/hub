package pos_api

import (
	"card/phoenix"
	"card/util"
	"encoding/json"
	"net/http"

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

	reqToken := r.Header.Get("Authorization")
	log.Info("auth token : ", reqToken)

	var req AddInvoiceRequest

	// err := decodeJSONBody(w, r, &req)
	// if err != nil {
	// 	var mr *malformedRequest
	// 	if errors.As(err, &mr) {
	// 		log.Error(mr.msg)
	// 	} else {
	// 		log.Error(err.Error())
	// 	}
	// 	return
	// }

	log.Info("AddInvoice : ", req)

	// create invoice on Phoenix server
	invoiceRequest := phoenix.ReceiveLightningPaymentRequest{
		Description: req.Memo,
		AmountSat:   req.Amt,
		ExternalId:  "ext_id",
	}
	invoice, err := phoenix.ReceiveLightningPayment(invoiceRequest)
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
	resp.Hash = "abcd"

	respJson, err := json.Marshal(resp)
	util.Check(err)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(respJson)
}
