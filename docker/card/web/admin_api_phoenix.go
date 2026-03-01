package web

import (
	"card/phoenix"
	"card/util"
	"net/http"

	log "github.com/sirupsen/logrus"
)

func (app *App) adminApiPhoenix(w http.ResponseWriter, r *http.Request) {
	type channelJSON struct {
		State             string `json:"state"`
		ChannelID         string `json:"channelId"`
		BalanceMsat       int64  `json:"balanceMsat"`
		InboundLiquidMsat int64  `json:"inboundLiquidMsat"`
	}

	balance, err := phoenix.GetBalance()
	if err != nil {
		log.Warn("phoenix balance error: ", err)
	}

	offer, err := phoenix.GetOffer()
	if err != nil {
		log.Warn("phoenix offer error: ", err)
	}

	channels, err := phoenix.ListChannels()
	if err != nil {
		log.Warn("phoenix channels error: ", err)
	}

	channelViews := make([]channelJSON, 0, len(channels))
	for _, ch := range channels {
		channelViews = append(channelViews, channelJSON{
			State:             ch.State,
			ChannelID:         ch.ChannelID,
			BalanceMsat:       ch.BalanceMsat,
			InboundLiquidMsat: ch.InboundLiquidMsat,
		})
	}

	writeJSON(w, map[string]interface{}{
		"balanceSat":   balance.BalanceSat,
		"feeCreditSat": balance.FeeCreditSat,
		"offer":        offer,
		"offerQr":      util.QrPngBase64Encode(offer),
		"channels":     channelViews,
	})
}
