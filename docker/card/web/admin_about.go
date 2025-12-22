package web

import (
	"card/build"
	"database/sql"
	"net/http"
)

func Admin_About(db_conn *sql.DB, w http.ResponseWriter, r *http.Request) {

	template_path := "/admin/about/index.html"

	data := struct {
		SwVersion   string
		SwBuildDate string
		SwBuildTime string
	}{
		SwVersion:   build.Version,
		SwBuildDate: build.Date,
		SwBuildTime: build.Time,
	}

	RenderHtmlFromTemplate(w, template_path, data)
}
