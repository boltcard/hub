package web

import (
	"net/http"
)

func (app *App) adminApiDatabaseDownload(w http.ResponseWriter, r *http.Request) {
	databaseDownload(w)
}

func (app *App) adminApiDatabaseImport(w http.ResponseWriter, r *http.Request) {
	databaseImport(w, r)
}
