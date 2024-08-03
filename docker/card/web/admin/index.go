package admin

import (
	"card/build"
	"card/phoenix"
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

	totalInboundSats := 0
	for _, channel := range info.Channels {
		totalInboundSats += channel.InboundLiquiditySat
	}

	// https://gosamples.dev/print-number-thousands-separator/
	// https://stackoverflow.com/questions/11123865/format-a-go-string-without-printing
	p := message.NewPrinter(language.English)
	FeeCreditSatStr := p.Sprintf("%d sats", balance.FeeCreditSat)
	BalanceSatStr := p.Sprintf("%d sats", balance.BalanceSat)
	TotalInboundSatsStr := p.Sprintf("%d sats", totalInboundSats)

	data := struct {
		FeeCredit   string
		Balance     string
		Inbound     string
		SwVersion   string
		SwBuildDate string
		SwBuildTime string
	}{
		FeeCredit:   FeeCreditSatStr,
		Balance:     BalanceSatStr,
		Inbound:     TotalInboundSatsStr,
		SwVersion:   build.Version,
		SwBuildDate: build.Date,
		SwBuildTime: build.Time,
	}

	web.RenderHtmlFromTemplate(w, template_path, data)
}
