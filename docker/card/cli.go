package main

import (
	"card/db"
	"card/phoenix"
	"card/util"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"time"

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
	case "WipeCard":
		wipeCard(db_conn, args)
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
// $ docker exec -it card bash
// # ./app SetupCardAmountForTag set1 10000
func setupCardAmountForTag(db_conn *sql.DB, args []string) {

	if len(args) < 3 {
		log.Warn("needs group_tag & amount_sats")
		return
	}

	groupTag := args[1]
	amountSats, err := strconv.Atoi(args[2])
	if err != nil {
		log.Error("invalid amount_sats: ", err)
		return
	}

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
// $ docker exec -it card bash
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
// $ docker exec -it card bash
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
	if err != nil {
		log.Error("invalid max_group_num: ", err)
		return
	}

	initialBalanceInt, err := strconv.Atoi(initialBalance)
	if err != nil {
		log.Error("invalid initial_balance: ", err)
		return
	}

	expiryHoursInt, err := strconv.Atoi(expiryHours)
	if err != nil {
		log.Error("invalid expiry_hours: ", err)
		return
	}

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

// used for testing the wipe card function
//
// $ docker exec -it card bash
// # ./app WipeCard card_id
func wipeCard(db_conn *sql.DB, args []string) {

	if len(args) != 2 {
		log.Warn("needs WipeCard card_id")
		return
	}

	cardIdStr := args[1]
	cardId, err := strconv.Atoi(cardIdStr)
	if err != nil {
		log.Warn("invalid card_id : ", cardIdStr)
		return
	}
	if cardId == 0 {
		log.Warn("card not found for id : ", cardId)
		return
	}

	card, err := db.Db_get_card(db_conn, cardId)
	if err != nil {
		log.Error("error getting card for id : ", cardId, " error : ", err)
		return
	}
	if card.Wiped == "Y" {
		log.Warn("card already wiped for id : ", cardId)
		return
	}

	log.Info("len(args) :", len(args))
	log.Info("cardId :", cardId)

	wipeData := struct {
		Version int    `json:"version"`
		Action  string `json:"action"`
		K0      string `json:"k0"`
		K1      string `json:"k1"`
		K2      string `json:"k2"`
		K3      string `json:"k3"`
		K4      string `json:"k4"`
	}{
		Version: 1,
		Action:  "wipe",
		K0:      card.Key0_auth,
		K1:      card.Key1_enc,
		K2:      card.Key2_cmac,
		K3:      card.Key3,
		K4:      card.Key4,
	}

	wipeDataJson, err := json.Marshal(wipeData)
	if err != nil {
		log.Error("error marshaling wipe data : ", err)
		return
	}

	fmt.Println("make this link into a QR code for URL")
	fmt.Println("e.g. with https://www.qrcode-monkey.com/#text")
	fmt.Println("and scan with your mobile device : ")
	fmt.Println(string(wipeDataJson))
}
