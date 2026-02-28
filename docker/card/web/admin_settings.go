package web

import (
	"database/sql"
	"net/http"
	"strings"

	"card/db"
)

func Admin_Settings(db_conn *sql.DB, w http.ResponseWriter, r *http.Request) {

	template_path := "/admin/settings/index.html"

	settings := db.Db_select_all_settings(db_conn)

	for i, s := range settings {
		if strings.HasSuffix(s.Name, "_hash") ||
			strings.HasSuffix(s.Name, "_token") ||
			strings.HasSuffix(s.Name, "_code") {
			settings[i].Value = "REDACTED"
		}
	}

	data := struct {
		Settings []db.Setting
	}{
		Settings: settings,
	}

	RenderHtmlFromTemplate(w, template_path, data)
}
