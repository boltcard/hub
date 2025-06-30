package web

import (
	"card/phoenix"
	"card/util"
	"database/sql"
	"net/http"
	"strconv"

	log "github.com/sirupsen/logrus"
)

func Admin2_Phoenix(db_conn *sql.DB, w http.ResponseWriter, r *http.Request) {

	template_path := "/admin2/phoenix/index.html"

	balance, err := phoenix.GetBalance()
	if err != nil {
		log.Warn("phoenix error: ", err.Error())
	}

	offer, err := phoenix.GetOffer()
	if err != nil {
		log.Warn("phoenix error: ", err.Error())
	}

	OfferQrPngEncoded := util.QrPngBase64Encode(offer)

	data := struct {
		BalanceSat        string
		FeeCreditSat      string
		OfferQrPngEncoded string
	}{
		BalanceSat:        strconv.Itoa(balance.BalanceSat),
		FeeCreditSat:      strconv.Itoa(balance.FeeCreditSat),
		OfferQrPngEncoded: OfferQrPngEncoded,
	}

	RenderHtmlFromTemplate(w, template_path, data)
}
