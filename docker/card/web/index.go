package web

import (
	"net/http"
)

// domain home page
func HomePage(w http.ResponseWriter, r *http.Request) {

	template_path := "/index.html"
	RenderHtmlFromTemplate(w, template_path, nil)
}

// get card balance & transaction table
func BalancePage(w http.ResponseWriter, r *http.Request) {

	template_path := "/balance/index.html"
	RenderHtmlFromTemplate(w, template_path, nil)
}
