package pos_api

import (
	"card/util"
	"encoding/json"
	"net/http"

	log "github.com/sirupsen/logrus"
)

type UserInvoice struct {
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

type GetUserInvoicesResponse []UserInvoice

func GetUserInvoices(w http.ResponseWriter, r *http.Request) {
	log.Info("pos_api GetUserInvoices request received")

	// using Phoenix Server GetIncomingPayments /payments/incoming?externalId={externalId} crashes Phoenix May 2024
	// seems to be when there are a lot of incoming payments with the same externalId (maybe >100)

	var resp GetUserInvoicesResponse

	var userInvoice1 UserInvoice
	userInvoice1.PaymentRequest = "1"

	var userInvoice2 UserInvoice
	userInvoice2.PaymentRequest = "2"

	resp = append(resp, userInvoice1)
	resp = append(resp, userInvoice2)

	respJson, err := json.Marshal(resp)
	util.Check(err)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(respJson)
}
