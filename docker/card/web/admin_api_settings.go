package web

import (
	"card/db"
	"encoding/json"
	"net/http"
	"strings"

	log "github.com/sirupsen/logrus"
)

func (app *App) adminApiGetSettings(w http.ResponseWriter, r *http.Request) {
	settings := db.Db_select_all_settings(app.db_conn)

	type settingJSON struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	}

	result := make([]settingJSON, 0, len(settings))
	logLevel := ""
	for _, s := range settings {
		value := s.Value
		if strings.HasSuffix(s.Name, "_hash") ||
			strings.HasSuffix(s.Name, "_token") ||
			strings.HasSuffix(s.Name, "_code") {
			value = "REDACTED"
		}
		if s.Name == "log_level" {
			logLevel = s.Value
		}
		result = append(result, settingJSON{Name: s.Name, Value: value})
	}

	writeJSON(w, map[string]interface{}{
		"settings": result,
		"logLevel": logLevel,
		"logLevels": []string{
			"panic", "fatal", "error", "warn", "info", "debug", "trace",
		},
	})
}

func (app *App) adminApiSetLogLevel(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Level string `json:"level"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, map[string]string{"error": "invalid request body"})
		return
	}

	valid := map[string]bool{
		"panic": true, "fatal": true, "error": true,
		"warn": true, "info": true, "debug": true, "trace": true,
	}
	if !valid[req.Level] {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, map[string]string{"error": "invalid log level"})
		return
	}

	db.Db_set_setting(app.db_conn, "log_level", req.Level)

	lvl, err := log.ParseLevel(req.Level)
	if err == nil {
		log.SetLevel(lvl)
	}

	writeJSON(w, map[string]bool{"ok": true})
}
