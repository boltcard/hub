package web

import (
	"card/phoenix"
	"card/util"
	"database/sql"
	"net/http"
	"strconv"

	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"
)

func msatToSatStr(msat int64) string {
	return decimal.NewFromInt(msat).Div(decimal.NewFromInt(1000)).String() + " sat"
}

type ChannelInfo struct {
	State         string
	ChannelID     string
	Balance       string
	InboundLiquid string
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
			State:         ch.State,
			ChannelID:     ch.ChannelID,
			Balance:       msatToSatStr(ch.BalanceMsat),
			InboundLiquid: msatToSatStr(ch.InboundLiquidMsat),
		})
	}

	data := struct {
		Balance           string
		FeeCredit         string
		OfferQrPngEncoded string
		Channels          []ChannelInfo
	}{
		Balance:           strconv.Itoa(balance.BalanceSat) + " sat",
		FeeCredit:         strconv.Itoa(balance.FeeCreditSat) + " sat",
		OfferQrPngEncoded: OfferQrPngEncoded,
		Channels:          channels,
	}

	RenderHtmlFromTemplate(w, template_path, data)
}
