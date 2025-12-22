package web

import (
	"database/sql"
	"net/http"
)

func Admin_Settings(db_conn *sql.DB, w http.ResponseWriter, r *http.Request) {

	template_path := "/admin/settings/index.html"

	data := struct {
	}{}

	RenderHtmlFromTemplate(w, template_path, data)
}
