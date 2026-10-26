package web

import (
	"card/db"
	"encoding/json"
	"net/http"
	"os"

	log "github.com/sirupsen/logrus"
)

// adminApiDatabaseDownload returns the whole database, which holds every
// card's keys (enough to forge taps and spend any card's balance), so it
// re-checks the admin password on top of the session cookie. POST with a
// JSON body {"password": "..."}.
func (app *App) adminApiDatabaseDownload(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, map[string]string{"error": "invalid request body"})
		return
	}

	if !app.verifyAdminPassword(req.Password) {
		log.Warn("databaseDownload: rejected, invalid admin password")
		w.WriteHeader(http.StatusForbidden)
		writeJSON(w, map[string]string{"error": "invalid password"})
		return
	}

	log.Info("databaseDownload: database downloaded by authenticated admin")
	databaseDownload(w)
}

// adminApiDatabaseImport replaces the whole database, including card
// balances and the admin password hash, so it re-checks the admin password
// on top of the session cookie (sent as the "password" multipart field).
func (app *App) adminApiDatabaseImport(w http.ResponseWriter, r *http.Request) {
	// 50 MB max, matching databaseImport (which then reuses the parsed form)
	if err := r.ParseMultipartForm(50 << 20); err != nil {
		http.Error(w, "invalid upload", http.StatusBadRequest)
		return
	}

	if !app.verifyAdminPassword(r.FormValue("password")) {
		log.Warn("databaseImport: rejected, invalid admin password")
		http.Error(w, "invalid password", http.StatusForbidden)
		return
	}

	databaseImport(w, r)
}

func (app *App) adminApiDatabaseStats(w http.ResponseWriter, r *http.Request) {
	stats := make(map[string]interface{})

	info, err := os.Stat("/card_data/cards.db")
	if err != nil {
		log.Warn("database stats: ", err)
	} else {
		stats["fileSizeBytes"] = info.Size()
	}

	stats["schemaVersion"] = db.Db_get_setting(app.db_read, "schema_version_number")

	counts, err := db.Db_get_table_counts(app.db_read)
	if err != nil {
		log.Warn("database stats table counts: ", err)
	} else {
		stats["tables"] = counts
	}

	writeJSON(w, stats)
}
