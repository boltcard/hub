package web

import (
	"card/db"
	"card/phoenix"
	"net/http"
	"strconv"
	"time"

	decodepay "github.com/nbd-wtf/ln-decodepay"
	log "github.com/sirupsen/logrus"
)

func (app *App) CreateHandler_LnurlwCallback() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		log.Info("LnurlwCallback received")

		// TODO: check host domain

		// check k1 value is valid
		param_k1 := r.URL.Query().Get("k1")
		if param_k1 == "" {
			w.Write([]byte(`{"status": "ERROR", "reason": "k1 not found"}`))
			return
		}

		cardId, lnurlwK1Expiry := db.Db_get_lnurlw_k1(app.db_conn, param_k1)
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
		param_pr := r.URL.Query().Get("pr")
		bolt11, _ := decodepay.Decodepay(param_pr)
		amountSats := int(bolt11.MSatoshi / 1000)

		// detect gift card use, i.e. sweeping of max_withdraw_sats amount
		maxWithdrawableSats, _ := strconv.Atoi(db.Db_get_setting(app.db_conn, "max_withdraw_sats"))
		if amountSats == maxWithdrawableSats {
			w.Write([]byte(`{"status": "ERROR", "reason": "this is a bolt card - use a Point of Sale"}`))
			return
		}

		// check the card balance
		total_paid_receipts := db.Db_get_total_paid_receipts(app.db_conn, cardId)
		total_paid_payments := db.Db_get_total_paid_payments(app.db_conn, cardId)
		total_card_balance := total_paid_receipts - total_paid_payments

		// calculate the max network fee possible
		// https://phoenix.acinq.co/faq#what-are-the-fees
		// Sending via Lightning 0.4 % + 4 sat

		max_network_fee_sats := 4 + amountSats*4/1000

		log.Info("amountSats ", amountSats)
		log.Info("max_network_fee_sats ", max_network_fee_sats)
		log.Info("total_card_balance ", total_card_balance)

		if amountSats > total_card_balance {
			log.Info("amountSats > total_card_balance : insufficient funds on card")
			w.Write([]byte(`{"status": "ERROR", "reason": "Insufficient funds"}`))
			return
		}

		if amountSats+max_network_fee_sats > total_card_balance {
			log.Info("amountSats + max_network_fee_sats > total_card_balance : insufficient funds on card")
			w.Write([]byte(`{"status": "ERROR", "reason": "Insufficient funds with network fees"}`))
			return
		}

		// save the lightning invoice
		card_payment_id := db.Db_add_card_payment(app.db_conn, cardId, amountSats, param_pr)
		// log.Info("card_payment_id ", card_payment_id)

		// TODO: check the payment rules (max withdrawal amount, max per day, PIN number)

		// make payment
		var payInvoiceRequest phoenix.SendLightningPaymentRequest

		payInvoiceRequest.Invoice = param_pr
		payInvoiceRequest.AmountSat = strconv.Itoa(amountSats)

		log.Info("attempting payment")
		payInvoiceResponse, payInvoiceResult, err := phoenix.SendLightningPayment(payInvoiceRequest)

		if err != nil {
			log.Error(err)
		}

		log.Info("payInvoiceResult : ", payInvoiceResult)
		log.Info("payInvoiceResponse ", payInvoiceResponse)

		switch payInvoiceResult {
		case "no_config":
			log.Error("phoenix config not set, card_payment_id = ", card_payment_id)
			w.Write([]byte(`{"status": "ERROR", "reason": "phoenix config not set"}`))
			log.Error("unlock funds, card_payment_id = ", card_payment_id)
			db.Db_update_card_payment_unpaid(app.db_conn, card_payment_id)
			return
		case "failed_request_creation":
			log.Error("failed to create a request for SendLightningPayment, card_payment_id = ", card_payment_id)
			w.Write([]byte(`{"status": "ERROR", "reason": "request creation failed"}`))
			log.Error("unlock funds, card_payment_id = ", card_payment_id)
			db.Db_update_card_payment_unpaid(app.db_conn, card_payment_id)
			return
		case "phoenix_api_timeout":
			log.Error("there was a timeout on the Phoenix API, card_payment_id = ", card_payment_id)
			w.Write([]byte(`{"status": "ERROR", "reason": "phoenix api timeout"}`))
			// don't unlock funds - must be handled manually
			return
		case "failed_read_response":
			log.Error("failed to create a request for SendLightningPayment, card_payment_id = ", card_payment_id)
			w.Write([]byte(`{"status": "ERROR", "reason": "request creation failed"}`))
			log.Error("unlock funds, card_payment_id = ", card_payment_id)
			db.Db_update_card_payment_unpaid(app.db_conn, card_payment_id)
			return
		case "fail_status_code":
			log.Error("the phoenix API response status was a fail, card_payment_id = ", card_payment_id)
			w.Write([]byte(`{"status": "ERROR", "reason": "phoenix api status fail"}`))
			// TODO: check this re. funds, currently left locked
			return
		case "failed_decode_response":
			log.Error("failed to decode the response from Phoenix API, card_payment_id = ", card_payment_id)
			w.Write([]byte(`{"status": "ERROR", "reason": "phoenix api decode failed"}`))
			// don't unlock funds - must be handled manually
			return
		case "no_error":
			// continue on possible successful payment path
			break
		default:
			// should never happen
			log.Error("payInvoiceResult is invalid, card_payment_id = ", card_payment_id)
			w.Write([]byte(`{"status": "ERROR", "reason": "bad payInvoiceResult"}`))
			// don't unlock funds - must be handled manually
			return
		}

		switch payInvoiceResponse.Reason {
		case "this invoice has already been paid":
			log.Error("duplicate invoice presented, card_payment_id = ", card_payment_id)
			w.Write([]byte(`{"status": "ERROR", "reason": "duplicate invoice"}`))
			log.Error("unlock funds, card_payment_id = ", card_payment_id)
			db.Db_update_card_payment_unpaid(app.db_conn, card_payment_id)
			return
		case "recipient node rejected the payment":
			log.Error("payment was rejected, card_payment_id = ", card_payment_id)
			w.Write([]byte(`{"status": "ERROR", "reason": "receiver rejected"}`))
			log.Error("unlock funds, card_payment_id = ", card_payment_id)
			db.Db_update_card_payment_unpaid(app.db_conn, card_payment_id)
			return
		case "not enough funds in wallet to afford payment":
			log.Error("funds too low, card_payment_id = ", card_payment_id)
			w.Write([]byte(`{"status": "ERROR", "reason": "funds low"}`))
			log.Error("unlock funds, card_payment_id = ", card_payment_id)
			db.Db_update_card_payment_unpaid(app.db_conn, card_payment_id)
			return
		case "routing fees are insufficient":
			log.Error("fees low or route not found, card_payment_id = ", card_payment_id)
			w.Write([]byte(`{"status": "ERROR", "reason": "route not found"}`))
			log.Error("unlock funds, card_payment_id = ", card_payment_id)
			db.Db_update_card_payment_unpaid(app.db_conn, card_payment_id)
			return
		case "":
			// continue on possible successful payment path
			break
		default:
			// should never happen
			log.Error("phoenix result is invalid, card_payment_id = ", card_payment_id)
			w.Write([]byte(`{"status": "ERROR", "reason": "bad phoenix result"}`))
			// don't unlock funds - must be handled manually
			return
		}

		//TODO: can check with Phoenix API call : get-outgoing-payment PaymentId

		// update card_payment record to add payInvoiceResponse.RoutingFeeSat
		db.Db_update_card_payment_fee(app.db_conn, card_payment_id, payInvoiceResponse.RoutingFeeSat)

		// send response
		jsonData := []byte(`{"status":"OK"}`)
		w.Write(jsonData)
	}
}
