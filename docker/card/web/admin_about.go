package web

import (
	"card/build"
	"card/phoenix"
	"database/sql"
	"net/http"

	log "github.com/sirupsen/logrus"
)

func Admin_About(db_conn *sql.DB, w http.ResponseWriter, r *http.Request) {

	if r.Method == "POST" {
		adminAboutPost(w, r)
		return
	}

	template_path := "/admin/about/index.html"

	phoenixdVersion := ""
	nodeInfo, err := phoenix.GetNodeInfo()
	if err != nil {
		log.Warn("Admin_About: failed to get phoenixd version: ", err.Error())
	} else {
		phoenixdVersion = nodeInfo.Version
	}

	latestVersion := CheckLatestVersion()
	updateAvailable := false
	if latestVersion != "" {
		updateAvailable = CompareVersions(build.Version, latestVersion) > 0
	}

	data := struct {
		SwVersion       string
		SwBuildDate     string
		SwBuildTime     string
		PhoenixdVersion string
		LatestVersion   string
		UpdateAvailable bool
	}{
		SwVersion:       build.Version,
		SwBuildDate:     build.Date,
		SwBuildTime:     build.Time,
		PhoenixdVersion: phoenixdVersion,
		LatestVersion:   latestVersion,
		UpdateAvailable: updateAvailable,
	}

	RenderHtmlFromTemplate(w, template_path, data)
}

func adminAboutPost(w http.ResponseWriter, r *http.Request) {

	action := r.FormValue("action")

	if action != "update" {
		http.Redirect(w, r, "/admin/about/", http.StatusSeeOther)
		return
	}

	err := TriggerUpdate()
	if err != nil {
		log.Error("TriggerUpdate failed: ", err)
		template_path := "/admin/about/updating.html"
		data := struct {
			Error string
		}{
			Error: err.Error(),
		}
		RenderHtmlFromTemplate(w, template_path, data)
		return
	}

	template_path := "/admin/about/updating.html"
	data := struct {
		Error string
	}{
		Error: "",
	}
	RenderHtmlFromTemplate(w, template_path, data)
}
