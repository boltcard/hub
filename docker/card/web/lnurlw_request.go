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

		cardMatch, cardId, ctr := Find_card(app.db_conn, p, c)

		if !cardMatch {
			log.Info("card not found")
			w.Write([]byte(`{"status": "ERROR", "reason": "card not found"}`))
			return
		}

		log.Info("card_id = " + strconv.Itoa(cardId))

		// check counter is incremented
		cardLastCounter := db.Db_get_card_counter(app.db_conn, cardId)
		if ctr <= cardLastCounter {
			log.Info("card counter not incremented")
			w.Write([]byte(`{"status": "ERROR", "reason": "card counter not incremented"}`))
			return
		}

		// store new counter value
		db.Db_set_card_counter(app.db_conn, cardId, ctr)

		// create and store lnurlw_k1
		lnurlwK1 := util.Random_hex()
		lnurlwK1Expiry := time.Now().Unix() + 10 // TODO: get timeout setting
		db.Db_set_lnurlw_k1(app.db_conn, cardId, lnurlwK1, lnurlwK1Expiry)

		// prepare response
		var resObj LnurlwResponse

		minWithdrawableSats, _ := strconv.Atoi(db.Db_get_setting(app.db_conn, "min_withdraw_sats"))
		maxWithdrawableSats, _ := strconv.Atoi(db.Db_get_setting(app.db_conn, "max_withdraw_sats"))

		resObj.Tag = "withdrawRequest"
		resObj.Callback = "https://" + db.Db_get_setting(app.db_conn, "host_domain") + "/cb"
		resObj.Lnurlwk1 = lnurlwK1
		resObj.MinWithdrawable = minWithdrawableSats * 1000
		resObj.MaxWithdrawable = maxWithdrawableSats * 1000

		log.Info("sending response for lnurlw request")

		writeJSON(w, resObj)
	}
}
