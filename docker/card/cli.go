package main

import (
	"card/db"
	"card/phoenix"
	"card/util"
	"database/sql"
	"fmt"
	"net/url"
	"strconv"
	"time"

	_ "github.com/mattn/go-sqlite3"
	log "github.com/sirupsen/logrus"
)

func processArgs(db_conn *sql.DB, args []string) {

	switch args[0] {
	case "SendLightningPayment":
		sendLightningPayment(args)
	case "ClearCardBalancesForTag":
		clearCardBalancesForTag(db_conn, args)
	case "SetupCardAmountForTag":
		setupCardAmountForTag(db_conn, args)
	case "ProgramBatch":
		programBatch(db_conn, args)
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
func setupCardAmountForTag(db_conn *sql.DB, args []string) {

	if len(args) < 3 {
		log.Warn("needs group_tag & amount_sats")
		return
	}

	groupTag := args[1]
	amountSats, err := strconv.Atoi(args[2])
	util.CheckAndPanic(err)

	cards := db.Db_select_cards_with_group_tag(db_conn, groupTag)

	for _, card := range cards {

		receipts := db.Db_get_total_paid_receipts(db_conn, card.CardId)

		if receipts > 0 {
			log.Error("unexpected card receipts for cardId ", card.CardId)
			return
		}

		// a unique payment_hash is needed so we put a unique description in here
		// there is expected to be only one loading per card
		card_receipt_id := db.Db_add_card_receipt(db_conn, card.CardId, "", strconv.Itoa(card.CardId), amountSats)
		db.Db_update_receipt_paid(db_conn, card_receipt_id)
	}

	log.Info("card setup has been successful for group : ", groupTag)
}

// used for clearing down balances after events
//
// $ docker container ps
// $ docker exec -it ContainerId sh
// # ./app ClearCardBalancesForTag set1
func clearCardBalancesForTag(db_conn *sql.DB, args []string) {

	if len(args) < 2 {
		log.Warn("needs group_tag")
		return
	}

	groupTag := args[1]

	cards := db.Db_select_cards_with_group_tag(db_conn, groupTag)

	for _, card := range cards {

		balance := getBalance(db_conn, card.CardId)

		if balance > 0 {
			log.Info("card.CardId : ", card.CardId)
			log.Info("balance : ", balance)

			// add the reducing payment record with the current timestamp
			db.Db_add_card_payment(db_conn, card.CardId, balance, "")

			balance = getBalance(db_conn, card.CardId)

			// verify that the card balance is <= 0
			if balance > 0 {
				log.Error("unexpected card balance > 0 for cardId ", card.CardId)
				return
			}
		}
	}

	log.Info("card balances have been successfully cleared for group : ", groupTag)
}

func getBalance(db_conn *sql.DB, cardId int) int {
	// get all transactions on the card
	txs := db.Db_select_card_txs(db_conn, cardId)

	// calculate the card balance
	balance := 0
	for _, tx := range txs {
		balance += tx.AmountSats + tx.FeeSats
	}

	return balance
}

// used for programming up cards in a batch
//
// $ docker container ps
// $ docker exec -it ContainerId sh
// # ./app ProgramBatch group_tag max_group_num initial_balance expiry_hours
func programBatch(db_conn *sql.DB, args []string) {

	if len(args) != 5 {
		log.Warn("needs ProgramBatch group_tag max_group_num initial_balance expiry_hours")
		return
	}

	groupTag := args[1]
	maxGroupNum := args[2]
	initialBalance := args[3]
	expiryHours := args[4]

	log.Info("len(args) :", len(args))
	log.Info("groupTag :", groupTag)
	log.Info("maxGroupNum :", maxGroupNum)
	log.Info("initialBalance :", initialBalance)
	log.Info("expiryHours :", expiryHours)

	// insert program_cards record

	secret := util.Random_hex()

	maxGroupNumInt, err := strconv.Atoi(maxGroupNum)
	util.CheckAndPanic(err)

	initialBalanceInt, err := strconv.Atoi(initialBalance)
	util.CheckAndPanic(err)

	expiryHoursInt, err := strconv.Atoi(expiryHours)
	util.CheckAndPanic(err)

	createTime := int(time.Now().Unix())
	expireTime := createTime + expiryHoursInt*60*60

	db.Db_insert_program_cards(db_conn, secret, groupTag, maxGroupNumInt, initialBalanceInt, createTime, expireTime)

	programUrl := `https://` + db.Db_get_setting(db_conn, "host_domain") + `/batch?s=` + secret
	boltcardLink := "boltcard://program?url=" + url.QueryEscape(programUrl)

	// show a boltcard://program?url=https%3A%2F%2F... link

	fmt.Println("make this link into a QR code for URL")
	fmt.Println("e.g. with https://www.qrcode-monkey.com/#url")
	fmt.Println("and scan with your mobile device : ")
	fmt.Println(boltcardLink)
}
