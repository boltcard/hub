package admin

import (
	"card/build"
	"card/db"
	"card/phoenix"
	"card/util"
	"card/web"
	"net/http"

	log "github.com/sirupsen/logrus"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

func Index(w http.ResponseWriter, r *http.Request) {

	template_path := "/dist/pages/admin/index.html"

	balance, err := phoenix.GetBalance()
	if err != nil {
		log.Warn("phoenix error: ", err.Error())
	}

	info, err := phoenix.GetNodeInfo()
	if err != nil {
		log.Warn("phoenix error: ", err.Error())
	}

	offer, err := phoenix.GetOffer()
	if err != nil {
		log.Warn("phoenix error: ", err.Error())
	}

	// log.Info("offer: ", offer)

	totalInboundSats := 0
	for _, channel := range info.Channels {
		totalInboundSats += channel.InboundLiquiditySat
	}

	// https://gosamples.dev/print-number-thousands-separator/
	// https://stackoverflow.com/questions/11123865/format-a-go-string-without-printing
	p := message.NewPrinter(language.English)
	FeeCreditSatStr := p.Sprintf("%d sats", balance.FeeCreditSat)
	BalanceSatStr := p.Sprintf("%d sats", balance.BalanceSat)
	ChannelsStr := p.Sprintf("%d", len(info.Channels))
	TotalInboundSatsStr := p.Sprintf("%d sats", totalInboundSats)

	OfferQrPngEncoded := util.QrPngBase64Encode(offer)

	// create LNURLw one time code
	lnurlw_token := util.Random_hex()
	db.Db_set_setting("lnurlw_token", lnurlw_token)
	LnurlwLink := "lnurlw://" + db.Db_get_setting("host_domain") + "/lnurlw?token=" + lnurlw_token
	LnurlwQrPngEncoded := util.QrPngBase64Encode(LnurlwLink)

	data := struct {
		FeeCredit         string
		Balance           string
		Channels          string
		Inbound           string
		OfferQrPngEncoded string
		LnurlwQr          string
		SwVersion         string
		SwBuildDate       string
		SwBuildTime       string
	}{
		FeeCredit:         FeeCreditSatStr,
		Balance:           BalanceSatStr,
		Channels:          ChannelsStr,
		Inbound:           TotalInboundSatsStr,
		OfferQrPngEncoded: OfferQrPngEncoded,
		LnurlwQr:          LnurlwQrPngEncoded,
		SwVersion:         build.Version,
		SwBuildDate:       build.Date,
		SwBuildTime:       build.Time,
	}

	web.RenderHtmlFromTemplate(w, template_path, data)
}
