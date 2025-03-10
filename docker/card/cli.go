package main

import (
	"card/phoenix"

	_ "github.com/mattn/go-sqlite3"
	log "github.com/sirupsen/logrus"
)

func processArgs(args []string) {

	switch args[0] {
	case "SendLightningPayment":
		sendLightningPayment(args)
	case "ClearCardBalancesForTag":
		clearCardBalancesForTag(args)
	default:
		log.Warn("CLI command not found : " + args[0])
	}
}

// for testing in a similar way to how it is called from LnurlwCallback
// ./app SendLightningPayment Invoice AmountSat
func sendLightningPayment(args []string) {
	var payInvoiceRequest phoenix.SendLightningPaymentRequest

	payInvoiceRequest.Invoice = args[1]
	payInvoiceRequest.AmountSat = args[2]

	payInvoiceResponse, payInvoiceResult, err := phoenix.SendLightningPayment(payInvoiceRequest)

	if err != nil {
		log.Error("Phoenix error response : ", err)
	}

	log.Info("payInvoiceResult : ", payInvoiceResult)
	log.Info("payInvoiceResponse : ", payInvoiceResponse)

	if payInvoiceResponse.PaymentId == "" {
		log.Info("no PaymentId") // might still be paid if timeout
	}

	// TODO:
	// get-outgoing-payment PaymentId
	// and store in the database
}

// used for clearing down balances after events
func clearCardBalancesForTag(args []string) {
}
