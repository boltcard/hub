package web

import (
	"net/http"
)

func Admin2_Index(w http.ResponseWriter, r *http.Request) {

	template_path := "/admin2/index.html"

	data := struct {
		NumCards string
	}{
		NumCards: "123",
	}

	RenderHtmlFromTemplate(w, template_path, data)
}
