package web

import (
	"database/sql"
	"net/http"
)

func Admin2_Settings(db_conn *sql.DB, w http.ResponseWriter, r *http.Request) {

	template_path := "/admin2/settings/index.html"

	data := struct {
	}{}

	RenderHtmlFromTemplate(w, template_path, data)
}
