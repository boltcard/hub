package wallet_api

import (
	"card/db"
	"card/util"

	"card/phoenix"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	log "github.com/sirupsen/logrus"
)

type UserInvoice struct {
	RHash struct {
		Type string `json:"type"`
		Data []int  `json:"data"`
	} `json:"r_hash"`
	PaymentRequest string `json:"payment_request"`
	AddIndex       string `json:"add_index"`
	PayReq         string `json:"pay_req"`
	Description    string `json:"description"`
	PaymentHash    string `json:"payment_hash"`
	IsPaid         bool   `json:"ispaid,omitempty"`
	Amt            int    `json:"amt"`
	ExpireTime     int    `json:"expire_time"`
	Timestamp      int    `json:"timestamp"`
	Type           string `json:"type"`
}

type UserInvoicesResponse []UserInvoice

func updateInvoiceStatus(paymentHash string) {
	//log.Info("update invoice for paymentHash ", paymentHash)

	// get status from phoenix server
	incomingPayment, err := phoenix.GetIncomingPayment(paymentHash)
	util.Check(err)

	//	log.Info("incomingPayment ", incomingPayment)

	// update status in the database if paid
	if incomingPayment.IsPaid {
		db.Db_set_receipt_paid(paymentHash)
	}
}

func GetUserInvoices(w http.ResponseWriter, r *http.Request) {
	log.Info("getUserInvoices request received")

	// set response header

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// get access_token

	authToken := r.Header.Get("Authorization")
	splitToken := strings.Split(authToken, "Bearer ")
	accessToken := splitToken[1]

	// get card_id from access_token

	card_id := db.Db_get_card_id_from_access_token(accessToken)

	if card_id == 0 {
		sendError(w, "Bad auth", 1, "no card found for access token")
		return
	}

	// get parameters

	limitProvided := true
	limitStr := r.URL.Query().Get("limit")
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limitProvided = false
	}

	//log.Info("limitProvided ", limitProvided)
	//log.Info("limit ", limit)

	// query database card receipts for card

	var cardReceipts db.CardReceipts

	if limitProvided {
		cardReceipts = db.Db_select_card_receipts_with_limit(card_id, limit)
	} else {
		cardReceipts = db.Db_select_card_receipts(card_id)
	}

	var resObj UserInvoicesResponse
	resObj = make([]UserInvoice, 0)
	var userInvoice UserInvoice

	for _, cardReceipt := range cardReceipts {
		userInvoice.PaymentRequest = cardReceipt.PaymentRequest
		userInvoice.AddIndex = "500" // lnd hub seems to do this
		userInvoice.RHash.Type = "buffer"
		userInvoice.RHash.Data = util.ConvertPaymentHash(cardReceipt.PaymentHash)
		userInvoice.PayReq = cardReceipt.PaymentRequest
		userInvoice.PaymentHash = cardReceipt.PaymentHash
		userInvoice.Description = ""
		userInvoice.IsPaid = false
		userInvoice.Amt = cardReceipt.AmountSats
		userInvoice.ExpireTime = cardReceipt.ExpireTime
		userInvoice.Timestamp = cardReceipt.Timestamp
		userInvoice.Type = "user_invoice"

		if cardReceipt.IsPaid == "Y" {
			userInvoice.IsPaid = true
		} else {
			updateInvoiceStatus(cardReceipt.PaymentHash)
			// userInvoice.IsPaid status will be updated on the next call
		}

		resObj = append(resObj, userInvoice)
	}

	resJson, err := json.Marshal(resObj)
	util.Check(err)

	//	log.Info("resJson string ", string(resJson))

	w.Write(resJson)
}
