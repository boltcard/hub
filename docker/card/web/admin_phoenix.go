package web

import (
	"card/phoenix"
	"card/util"
	"database/sql"
	"net/http"
	"strconv"

	log "github.com/sirupsen/logrus"
)

type ChannelInfo struct {
	State             string
	ChannelID         string
	BalanceMsat       string
	InboundLiquidMsat string
}

func Admin_Phoenix(db_conn *sql.DB, w http.ResponseWriter, r *http.Request) {

	template_path := "/admin/phoenix/index.html"

	balance, err := phoenix.GetBalance()
	if err != nil {
		log.Warn("phoenix error: ", err.Error())
	}

	offer, err := phoenix.GetOffer()
	if err != nil {
		log.Warn("phoenix error: ", err.Error())
	}

	phoenixChannels, err := phoenix.ListChannels()
	if err != nil {
		log.Warn("phoenix error: ", err.Error())
	}

	OfferQrPngEncoded := util.QrPngBase64Encode(offer)

	var channels []ChannelInfo
	for _, ch := range phoenixChannels {
		channels = append(channels, ChannelInfo{
			State:             ch.State,
			ChannelID:         ch.ChannelID,
			BalanceMsat:       strconv.FormatInt(ch.BalanceMsat, 10),
			InboundLiquidMsat: strconv.FormatInt(ch.InboundLiquidMsat, 10),
		})
	}

	data := struct {
		BalanceSat        string
		FeeCreditSat      string
		OfferQrPngEncoded string
		Channels          []ChannelInfo
	}{
		BalanceSat:        strconv.Itoa(balance.BalanceSat),
		FeeCreditSat:      strconv.Itoa(balance.FeeCreditSat),
		OfferQrPngEncoded: OfferQrPngEncoded,
		Channels:          channels,
	}

	RenderHtmlFromTemplate(w, template_path, data)
}
