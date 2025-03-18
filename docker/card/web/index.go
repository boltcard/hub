package web

import (
	"net/http"
)

func HomePage(w http.ResponseWriter, r *http.Request) {
	// TODO: return QR code for BOLT 12 Offer

	template_path := "/dist/pages/index.html"
	RenderHtmlFromTemplate(w, template_path, nil)
}

func BalancePage(w http.ResponseWriter, r *http.Request) {
	// get card balance & transaction table

	template_path := "/dist/pages/balance/index.html"
	RenderHtmlFromTemplate(w, template_path, nil)
}
