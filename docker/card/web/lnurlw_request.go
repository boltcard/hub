package web

import (
	"card/db"
	"card/util"

	"net/http"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"
)

type LnurlwResponse struct {
	Tag                 string `json:"tag"`
	Callback            string `json:"callback"`
	Lnurlwk1            string `json:"k1"`
	DefaultDescrription string `json:"default_description"`
	MinWithdrawable     int    `json:"minWithdrawable"`
	MaxWithdrawable     int    `json:"maxWithdrawable"`
	PinLimit            int    `json:"pinLimit,omitempty"`
	PayLink             string `json:"payLink,omitempty"`
}

func (app *App) CreateHandler_LnurlwRequest() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		log.Info("LnurlwRequest received")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		u := r.URL
		p, c, err := Get_p_c(u)
		if err != nil {
			w.Write([]byte(`{"status": "ERROR", "reason": "badly formatted request"}`))
			return
		}

		cardMatch, cardId, ctr := Find_card(app.db_read, p, c)

		if !cardMatch {
			log.Info("card not found")
			w.Write([]byte(`{"status": "ERROR", "reason": "card not found"}`))
			return
		}

		log.Info("card_id = " + strconv.Itoa(cardId))

		// check counter is incremented
		cardLastCounter := db.Db_get_card_counter(app.db_read, cardId)
		if ctr <= cardLastCounter {
			log.Info("card counter not incremented")
			w.Write([]byte(`{"status": "ERROR", "reason": "card counter not incremented"}`))
			return
		}

		// store new counter value
		db.Db_set_card_counter(app.db_write, cardId, ctr)

		// check card withdrawals are enabled
		lnurlwEnable := db.Db_get_card_lnurlw_enable(app.db_read, cardId)
		if lnurlwEnable != "Y" {
			log.Info("card withdrawals disabled")
			w.Write([]byte(`{"status": "ERROR", "reason": "withdrawals disabled"}`))
			return
		}

		// create and store lnurlw_k1
		lnurlwK1 := util.Random_hex()
		lnurlwK1Expiry := time.Now().Unix() + 10 // TODO: get timeout setting
		db.Db_set_lnurlw_k1(app.db_write, cardId, lnurlwK1, lnurlwK1Expiry)

		// prepare response
		var resObj LnurlwResponse

		minWithdrawableSats := 1
		maxWithdrawableSats := 100_000_000

		hostDomain := db.Db_get_setting(app.db_read, "host_domain")

		resObj.Tag = "withdrawRequest"
		resObj.Callback = "https://" + hostDomain + "/cb"
		resObj.Lnurlwk1 = lnurlwK1
		resObj.MinWithdrawable = minWithdrawableSats * 1000
		resObj.MaxWithdrawable = maxWithdrawableSats * 1000

		// Include payLink if enabled (LUD-19)
		if db.Db_get_card_pay_link_enabled(app.db_read, cardId) == "Y" {
			payLinkAddress := "pl." + util.Random_hex()[:8]
			expiryDays := 30 // default
			if v := db.Db_get_setting(app.db_read, "pay_link_expiry_days"); v != "" {
				if days, err := strconv.Atoi(v); err == nil && days > 0 {
					expiryDays = days
				}
			}
			db.Db_add_pay_link_address(app.db_write, payLinkAddress, cardId, expiryDays)
			resObj.PayLink = "https://" + hostDomain + "/.well-known/lnurlp/" + payLinkAddress
		}

		log.Info("sending response for lnurlw request")

		writeJSON(w, resObj)
	}
}
