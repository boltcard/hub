package web

import (
	"card/phoenix"
	"card/util"
	"encoding/json"
	"net/http"

	log "github.com/sirupsen/logrus"
)

type WalletApiUserInvoice struct {
	RHash          RHash  `json:"r_hash"`
	PaymentRequest string `json:"payment_request"`
	AddIndex       string `json:"add_index"`
	Description    string `json:"description"`
	PaymentHash    string `json:"payment_hash"`
	Ispaid         bool   `json:"ispaid"`
	Amt            int    `json:"amt"`
	ExpireTime     int    `json:"expire_time"`
	Timestamp      int    `json:"timestamp"`
	Type           string `json:"type"`
}

type GetUserInvoicesResponse []WalletApiUserInvoice

func (app *App) CreateHandler_PosApi_GetUserInvoices() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		log.Info("pos_api GetUserInvoices request received")

		// using Phoenix Server GetIncomingPayments /payments/incoming?externalId={externalId} crashes Phoenix May 2024
		// seems to be when there are a lot of incoming payments with the same externalId (maybe >100)
		// looks ok in v0.3.3 Sept 2024

		var resp GetUserInvoicesResponse

		pmt_list, err := phoenix.ListIncomingPayments(20, 0)
		if err != nil {
			log.Warn("phoenix error: ", err.Error())
		}

		var numCards = len(pmt_list)

		var userInvoice WalletApiUserInvoice
		for i := 0; i < numCards; i++ {
			userInvoice.PaymentRequest = pmt_list[i].Invoice
			userInvoice.PaymentHash = pmt_list[i].PaymentHash
			userInvoice.Ispaid = pmt_list[i].IsPaid
			userInvoice.Description = pmt_list[i].Description
			userInvoice.Amt = pmt_list[i].ReceivedSat
			//userInvoice.Timestamp

			resp = append(resp, userInvoice)
		}

		respJson, err := json.Marshal(resp)
		util.CheckAndPanic(err)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(respJson)
	}
}
