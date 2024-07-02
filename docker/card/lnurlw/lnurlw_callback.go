package lnurlw

import (
	"card/db"
	"card/phoenix"
	"card/util"
	"net/http"
	"strconv"
	"time"

	decodepay "github.com/nbd-wtf/ln-decodepay"
	log "github.com/sirupsen/logrus"
)

func LnurlwCallback(w http.ResponseWriter, req *http.Request) {

	log.Info("LnurlwCallback received")

	// TODO: check host domain

	// check k1 value is valid
	param_k1 := req.URL.Query().Get("k1")
	if param_k1 == "" {
		w.Write([]byte(`{"status": "ERROR", "reason": "k1 not found"}`))
		return
	}

	cardId, lnurlwK1Expiry := db.Db_get_lnurlw_k1(param_k1)
	if cardId == 0 {
		w.Write([]byte(`{"status": "ERROR", "reason": "card not found for k1 value"}`))
		return
	}

	// check k1 value is not timed out
	currentTime := time.Now().Unix()
	if uint64(currentTime) > lnurlwK1Expiry {
		w.Write([]byte(`{"status": "ERROR", "reason": "k1 value expired"}`))
		return
	}

	// decode the lightning invoice
	param_pr := req.URL.Query().Get("pr")
	bolt11, _ := decodepay.Decodepay(param_pr)
	amountSats := int(bolt11.MSatoshi / 1000)

	// save the lightning invoice
	db.Db_add_card_payment(cardId, amountSats, param_pr)

	// check the card balance
	total_paid_receipts := db.Db_get_total_paid_receipts(cardId)
	total_paid_payments := db.Db_get_total_payments(cardId)
	total_card_balance := total_paid_receipts - total_paid_payments

	// log.Info("amountSats ", amountSats)
	// log.Info("total_card_balance ", total_card_balance)

	if amountSats > total_card_balance {
		w.Write([]byte(`{"status": "ERROR", "reason": "card balance lower than payment amount"}`))
		return
	}

	// TODO: check the payment rules (max withdrawal amount, max per day, PIN number)

	// make payment
	var payInvoiceRequest phoenix.SendLightningPaymentRequest

	payInvoiceRequest.Invoice = param_pr
	payInvoiceRequest.AmountSat = strconv.Itoa(amountSats)

	payInvoiceResponse, err := phoenix.SendLightningPayment(payInvoiceRequest)
	util.Check(err)

	log.Info("payInvoiceResponse ", payInvoiceResponse)

	// send response
	jsonData := []byte(`{"status":"OK"}`)
	w.Write(jsonData)
}
