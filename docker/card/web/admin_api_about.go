package web

import (
	"card/build"
	"card/phoenix"
	"net/http"

	log "github.com/sirupsen/logrus"
)

func (app *App) adminApiAbout(w http.ResponseWriter, r *http.Request) {
	latestVersion := CheckLatestVersion()
	updateAvailable := false
	if latestVersion != "" {
		updateAvailable = CompareVersions(build.Version, latestVersion) == 1
	}

	phoenixdVersion := ""
	info, err := phoenix.GetNodeInfo()
	if err == nil {
		phoenixdVersion = info.Version
	} else {
		log.Warn("phoenix info error: ", err)
	}

	writeJSON(w, map[string]interface{}{
		"version":         build.Version,
		"buildDate":       build.Date,
		"buildTime":       build.Time,
		"phoenixdVersion": phoenixdVersion,
		"latestVersion":   latestVersion,
		"updateAvailable": updateAvailable,
	})
}

func (app *App) adminApiTriggerUpdate(w http.ResponseWriter, r *http.Request) {
	err := TriggerUpdate()
	if err != nil {
		log.Error("update trigger error: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, map[string]bool{"ok": true})
}
