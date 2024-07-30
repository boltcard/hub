package web

import (
	"card/phoenix"
	"encoding/base64"
	"net/http"

	log "github.com/sirupsen/logrus"
	"github.com/skip2/go-qrcode"
)

func HomePage(w http.ResponseWriter, r *http.Request) {
	// TODO: return QR code for BOLT 12 Offer

	template_path := "/dist/pages/index.html"
	RenderHtmlFromTemplate(w, template_path, nil)
}

func Bolt12Test(w http.ResponseWriter, r *http.Request) {
	// TODO: return QR code for BOLT 12 Offer

	template_path := "/dist/pages/bolt12/index.html"

	offer, err := phoenix.GetOffer()
	if err != nil {
		log.Warn("phoenix error: ", err.Error())
	}

	var offer_qr_png []byte
	offer_qr_png, err = qrcode.Encode(offer, qrcode.Medium, 256)
	if err != nil {
		log.Warn("qrcode error: ", err.Error())
	}

	// https://stackoverflow.com/questions/2807251/can-i-embed-a-png-image-into-an-html-page
	OfferQrPngEncoded := base64.StdEncoding.EncodeToString(offer_qr_png)

	data := struct {
		QrValue           string
		OfferQrPngEncoded string
	}{
		QrValue:           offer,
		OfferQrPngEncoded: OfferQrPngEncoded,
	}

	RenderHtmlFromTemplate(w, template_path, data)
}
