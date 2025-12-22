package web

import (
	"net/http"
)

func Admin_Index(w http.ResponseWriter, r *http.Request) {

	template_path := "/admin/index.html"

	data := struct {
		NumCards string
	}{
		NumCards: "123",
	}

	RenderHtmlFromTemplate(w, template_path, data)
}
