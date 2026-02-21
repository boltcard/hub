package web

import (
	"card/db"
	"card/phoenix"
	"database/sql"
	"net/http"
	"strconv"
	"time"

	decodepay "github.com/nbd-wtf/ln-decodepay"
	log "github.com/sirupsen/logrus"
)

// lnurlError writes an LNURL-compatible error JSON response.
func lnurlError(w http.ResponseWriter, reason string) {
	w.Write([]byte(`{"status": "ERROR", "reason": "` + reason + `"}`))
}

// handlePaymentResult processes the phoenix payment result code.
// Returns true if the payment failed and the handler should return.
func handlePaymentResult(w http.ResponseWriter, db_conn *sql.DB, payInvoiceResult string, card_payment_id int) bool {
	switch payInvoiceResult {
	case "no_config":
		log.Error("phoenix config not set, card_payment_id = ", card_payment_id)
		lnurlError(w, "phoenix config not set")
		db.Db_update_card_payment_unpaid(db_conn, card_payment_id)
		return true
	case "failed_request_creation":
		log.Error("failed to create a request for SendLightningPayment, card_payment_id = ", card_payment_id)
		lnurlError(w, "request creation failed")
		db.Db_update_card_payment_unpaid(db_conn, card_payment_id)
		return true
	case "phoenix_api_timeout":
		log.Error("there was a timeout on the Phoenix API, card_payment_id = ", card_payment_id)
		lnurlError(w, "phoenix api timeout")
		// don't unlock funds - must be handled manually
		return true
	case "failed_read_response":
		log.Error("failed to read response for SendLightningPayment, card_payment_id = ", card_payment_id)
		lnurlError(w, "request creation failed")
		db.Db_update_card_payment_unpaid(db_conn, card_payment_id)
		return true
	case "fail_status_code":
		log.Error("the phoenix API response status was a fail, card_payment_id = ", card_payment_id)
		lnurlError(w, "phoenix api status fail")
		// TODO: check this re. funds, currently left locked
		return true
	case "failed_decode_response":
		log.Error("failed to decode the response from Phoenix API, card_payment_id = ", card_payment_id)
		lnurlError(w, "phoenix api decode failed")
		// don't unlock funds - must be handled manually
		return true
	case "no_error":
		return false
	default:
		log.Error("payInvoiceResult is invalid, card_payment_id = ", card_payment_id)
		lnurlError(w, "bad payInvoiceResult")
		// don't unlock funds - must be handled manually
		return true
	}
}

// handlePaymentReason processes the phoenix payment reason string.
// Returns true if the payment failed and the handler should return.
func handlePaymentReason(w http.ResponseWriter, db_conn *sql.DB, reason string, card_payment_id int) bool {
	switch reason {
	case "this invoice has already been paid":
		log.Error("duplicate invoice presented, card_payment_id = ", card_payment_id)
		lnurlError(w, "duplicate invoice")
		db.Db_update_card_payment_unpaid(db_conn, card_payment_id)
		return true
	case "recipient node rejected the payment":
		log.Error("payment was rejected, card_payment_id = ", card_payment_id)
		lnurlError(w, "receiver rejected")
		db.Db_update_card_payment_unpaid(db_conn, card_payment_id)
		return true
	case "not enough funds in wallet to afford payment":
		log.Error("funds too low, card_payment_id = ", card_payment_id)
		lnurlError(w, "funds low")
		db.Db_update_card_payment_unpaid(db_conn, card_payment_id)
		return true
	case "routing fees are insufficient":
		log.Error("fees low or route not found, card_payment_id = ", card_payment_id)
		lnurlError(w, "route not found")
		db.Db_update_card_payment_unpaid(db_conn, card_payment_id)
		return true
	case "":
		return false
	default:
		log.Error("phoenix result is invalid, card_payment_id = ", card_payment_id)
		lnurlError(w, "bad phoenix result")
		// don't unlock funds - must be handled manually
		return true
	}
}

func (app *App) CreateHandler_LnurlwCallback() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		log.Info("LnurlwCallback received")

		// validate k1 parameter
		param_k1 := r.URL.Query().Get("k1")
		if param_k1 == "" {
			lnurlError(w, "k1 not found")
			return
		}

		cardId, lnurlwK1Expiry := db.Db_get_lnurlw_k1(app.db_conn, param_k1)
		if cardId == 0 {
			lnurlError(w, "card not found for k1 value")
			return
		}

		if uint64(time.Now().Unix()) > lnurlwK1Expiry {
			lnurlError(w, "k1 value expired")
			return
		}

		// decode and validate the lightning invoice
		param_pr := r.URL.Query().Get("pr")
		bolt11, err := decodepay.Decodepay(param_pr)
		if err != nil {
			log.Error("decodepay error: ", err)
			lnurlError(w, "invalid invoice")
			return
		}
		amountSats := int(bolt11.MSatoshi / 1000)

		// detect gift card use, i.e. sweeping of max_withdraw_sats amount
		maxWithdrawableSats, _ := strconv.Atoi(db.Db_get_setting(app.db_conn, "max_withdraw_sats"))
		if amountSats == maxWithdrawableSats {
			lnurlError(w, "this is a bolt card - use a Point of Sale")
			return
		}

		// check the card balance with fee headroom
		total_card_balance := db.Db_get_card_balance(app.db_conn, cardId)
		max_network_fee_sats := 4 + amountSats*4/1000 // Phoenix fee: 0.4% + 4 sat

		log.Info("amountSats ", amountSats)
		log.Info("max_network_fee_sats ", max_network_fee_sats)
		log.Info("total_card_balance ", total_card_balance)

		if amountSats > total_card_balance {
			log.Info("amountSats > total_card_balance : insufficient funds on card")
			lnurlError(w, "Insufficient funds")
			return
		}

		if amountSats+max_network_fee_sats > total_card_balance {
			log.Info("amountSats + max_network_fee_sats > total_card_balance : insufficient funds on card")
			lnurlError(w, "Insufficient funds with network fees")
			return
		}

		// reserve funds by inserting a payment record
		card_payment_id := db.Db_add_card_payment(app.db_conn, cardId, amountSats, param_pr)

		// TODO: check the payment rules (max withdrawal amount, max per day, PIN number)

		// execute the lightning payment
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

		// handle payment result and reason
		if handlePaymentResult(w, app.db_conn, payInvoiceResult, card_payment_id) {
			return
		}
		if handlePaymentReason(w, app.db_conn, payInvoiceResponse.Reason, card_payment_id) {
			return
		}

		// payment succeeded — record the routing fee
		db.Db_update_card_payment_fee(app.db_conn, card_payment_id, payInvoiceResponse.RoutingFeeSat)

		w.Write([]byte(`{"status":"OK"}`))
	}
}
