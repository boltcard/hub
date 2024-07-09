package admin

import (
	"card/build"
	"card/phoenix"
	"card/web"
	"encoding/base64"
	"net/http"

	log "github.com/sirupsen/logrus"
	qrcode "github.com/skip2/go-qrcode"
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

	var offer_qr_png []byte
	offer_qr_png, err = qrcode.Encode(offer, qrcode.Medium, 256)
	if err != nil {
		log.Warn("qrcode error: ", err.Error())
	}

	// https://stackoverflow.com/questions/2807251/can-i-embed-a-png-image-into-an-html-page
	OfferQrPngEncoded := base64.StdEncoding.EncodeToString(offer_qr_png)

	//TODO: create LNURLw one time code
	LnurlwQrPngEncoded := OfferQrPngEncoded

	data := struct {
		FeeCredit          string
		Balance            string
		Channels           string
		Inbound            string
		OfferQrPngEncoded  string
		LnurlwQrPngEncoded string
		SwVersion          string
		SwBuildDate        string
		SwBuildTime        string
	}{
		FeeCredit:          FeeCreditSatStr,
		Balance:            BalanceSatStr,
		Channels:           ChannelsStr,
		Inbound:            TotalInboundSatsStr,
		OfferQrPngEncoded:  OfferQrPngEncoded,
		LnurlwQrPngEncoded: LnurlwQrPngEncoded,
		SwVersion:          build.Version,
		SwBuildDate:        build.Date,
		SwBuildTime:        build.Time,
	}

	web.RenderHtmlFromTemplate(w, template_path, data)
}
