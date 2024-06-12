package web

import (
	"card/util"
	"html/template"
	"net/http"
)

var templates *template.Template

func InitTemplates() {
	templates = template.Must(template.ParseFiles(
		"/web-content/home.html",
		"/web-content/setup.html",
		"/web-content/login.html",
		"/web-content/dashboard.html",
	))
}

func renderTemplate(w http.ResponseWriter, tmpl string, data interface{}) {
	err := templates.ExecuteTemplate(w, tmpl+".html", data)
	util.Check(err)
}
