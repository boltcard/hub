package web

import (
	"database/sql"
	"net/http"
)

func Admin2_Cards(db_conn *sql.DB, w http.ResponseWriter, r *http.Request) {

	template_path := "/admin2/cards/index.html"

	data := struct {
	}{}

	RenderHtmlFromTemplate(w, template_path, data)
}
