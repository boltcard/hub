package web

import (
	"database/sql"
	"net/http"
	"strings"

	"card/db"

	log "github.com/sirupsen/logrus"
)

var validLogLevels = []string{"debug", "info", "warn", "error"}

func Admin_Settings(db_conn *sql.DB, w http.ResponseWriter, r *http.Request) {

	if r.Method == "POST" {
		adminSettingsPost(db_conn, w, r)
		return
	}

	template_path := "/admin/settings/index.html"

	settings := db.Db_select_all_settings(db_conn)

	currentLogLevel := ""
	for i, s := range settings {
		if s.Name == "log_level" {
			currentLogLevel = s.Value
		}
		if strings.HasSuffix(s.Name, "_hash") ||
			strings.HasSuffix(s.Name, "_token") ||
			strings.HasSuffix(s.Name, "_code") {
			settings[i].Value = "REDACTED"
		}
	}

	data := struct {
		Settings    []db.Setting
		LogLevel    string
		LogLevels   []string
	}{
		Settings:    settings,
		LogLevel:    currentLogLevel,
		LogLevels:   validLogLevels,
	}

	RenderHtmlFromTemplate(w, template_path, data)
}

func adminSettingsPost(db_conn *sql.DB, w http.ResponseWriter, r *http.Request) {

	logLevel := r.FormValue("log_level")

	valid := false
	for _, l := range validLogLevels {
		if logLevel == l {
			valid = true
			break
		}
	}

	if valid {
		db.Db_set_setting(db_conn, "log_level", logLevel)
		level, _ := log.ParseLevel(logLevel)
		log.SetLevel(level)
		log.Info("log level changed to ", logLevel)
	}

	http.Redirect(w, r, "/admin/settings/", http.StatusSeeOther)
}
