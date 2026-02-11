package web

import (
	"card/phoenix"
	"card/util"
	"database/sql"
	"net/http"
	"strconv"
	"strings"

	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"
)

func addCommas(s string) string {
	parts := strings.SplitN(s, ".", 2)
	intPart := parts[0]
	neg := ""
	if len(intPart) > 0 && intPart[0] == '-' {
		neg = "-"
		intPart = intPart[1:]
	}
	n := len(intPart)
	if n > 3 {
		var b strings.Builder
		for i, c := range intPart {
			if i > 0 && (n-i)%3 == 0 {
				b.WriteByte(',')
			}
			b.WriteRune(c)
		}
		intPart = b.String()
	}
	out := neg + intPart
	if len(parts) > 1 {
		out += "." + parts[1]
	}
	return out
}

func msatToSatStr(msat int64) string {
	return addCommas(decimal.NewFromInt(msat).Div(decimal.NewFromInt(1000)).String()) + " sat"
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
		Balance:           addCommas(strconv.Itoa(balance.BalanceSat)) + " sat",
		FeeCredit:         addCommas(strconv.Itoa(balance.FeeCreditSat)) + " sat",
		OfferQrPngEncoded: OfferQrPngEncoded,
		Channels:          channels,
	}

	RenderHtmlFromTemplate(w, template_path, data)
}
