package web

import (
	"card/build"
	"net/http"
)

func Admin2_Index(w http.ResponseWriter, r *http.Request) {

	template_path := "/admin2/index.html"

	data := struct {
		NumCards    string
		SwVersion   string
		SwBuildDate string
		SwBuildTime string
	}{
		NumCards:    "123",
		SwVersion:   build.Version,
		SwBuildDate: build.Date,
		SwBuildTime: build.Time,
	}

	RenderHtmlFromTemplate(w, template_path, data)
}
