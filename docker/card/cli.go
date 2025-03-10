package main

import (
	"card/db"
	"card/phoenix"
	"card/util"
	"strconv"

	_ "github.com/mattn/go-sqlite3"
	log "github.com/sirupsen/logrus"
)

func processArgs(args []string) {

	switch args[0] {
	case "SendLightningPayment":
		sendLightningPayment(args)
	case "ClearCardBalancesForTag":
		clearCardBalancesForTag(args)
	case "SetupCardAmountForTag":
		setupCardAmountForTag(args)
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

// used for setting up gift amounts for events
//
// $ docker container ps
// $ docker exec -it ContainerId sh
// # ./app SetupCardAmountForTag set1 10000
func setupCardAmountForTag(args []string) {

	if len(args) < 3 {
		log.Warn("needs group_tag & amount_sats")
	}

	groupTag := args[1]
	amountSats, err := strconv.Atoi(args[2])
	util.Check(err)

	cards := db.Db_select_cards_with_group_tag(groupTag)

	for _, card := range cards {

		receipts := db.Db_get_total_paid_receipts(card.CardId)

		if receipts > 0 {
			log.Error("unexpected card receipts for cardId ", card.CardId)
			return
		}

		// a unique payment_hash is needed so we put a unique description in here
		// there is expected to be only one loading per card
		card_receipt_id := db.Db_add_card_receipt(card.CardId, "", strconv.Itoa(card.CardId), amountSats)
		db.Db_update_receipt_paid(card_receipt_id)
	}

	log.Info("card setup has been successful for group : ", groupTag)
}

// used for clearing down balances after events
//
// $ docker container ps
// $ docker exec -it ContainerId sh
// # ./app ClearCardBalancesForTag set1
func clearCardBalancesForTag(args []string) {

	if len(args) < 2 {
		log.Warn("needs group_tag")
	}

	groupTag := args[1]

	cards := db.Db_select_cards_with_group_tag(groupTag)

	for _, card := range cards {

		balance := getBalance(card.CardId)

		if balance > 0 {
			log.Info("card.CardId : ", card.CardId)
			log.Info("balance : ", balance)

			// add the reducing payment record with the current timestamp
			db.Db_add_card_payment(card.CardId, balance, "")

			balance = getBalance(card.CardId)

			// verify that the card balance is <= 0
			if balance > 0 {
				log.Error("unexpected card balance > 0 for cardId ", card.CardId)
				return
			}
		}
	}

	log.Info("card balances have been successfully cleared for group : ", groupTag)
}

func getBalance(cardId int) int {
	// get all transactions on the card
	txs := db.Db_select_card_txs(cardId)

	// calculate the card balance
	balance := 0
	for _, tx := range txs {
		balance += tx.AmountSats + tx.FeeSats
	}

	return balance
}
