package web

import (
	"net/http"
)

func (app *App) adminApiListCards(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]interface{}{
		"cards": []interface{}{},
	})
}

func (app *App) adminApiCardRouter(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
	writeJSON(w, map[string]string{"error": "not implemented"})
}
