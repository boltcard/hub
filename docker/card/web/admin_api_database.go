package web

import (
	"card/db"
	"net/http"
	"os"

	log "github.com/sirupsen/logrus"
)

func (app *App) adminApiDatabaseDownload(w http.ResponseWriter, r *http.Request) {
	databaseDownload(w)
}

func (app *App) adminApiDatabaseImport(w http.ResponseWriter, r *http.Request) {
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

	stats["schemaVersion"] = db.Db_get_setting(app.db_conn, "schema_version_number")

	counts, err := db.Db_get_table_counts(app.db_conn)
	if err != nil {
		log.Warn("database stats table counts: ", err)
	} else {
		stats["tables"] = counts
	}

	writeJSON(w, stats)
}
