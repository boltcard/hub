package web

import (
	"card/build"
	"card/phoenix"
	"database/sql"
	"net/http"

	log "github.com/sirupsen/logrus"
)

func Admin_About(db_conn *sql.DB, w http.ResponseWriter, r *http.Request) {

	template_path := "/admin/about/index.html"

	phoenixdVersion := ""
	nodeInfo, err := phoenix.GetNodeInfo()
	if err != nil {
		log.Warn("Admin_About: failed to get phoenixd version: ", err.Error())
	} else {
		phoenixdVersion = nodeInfo.Version
	}

	data := struct {
		SwVersion       string
		SwBuildDate     string
		SwBuildTime     string
		PhoenixdVersion string
	}{
		SwVersion:       build.Version,
		SwBuildDate:     build.Date,
		SwBuildTime:     build.Time,
		PhoenixdVersion: phoenixdVersion,
	}

	RenderHtmlFromTemplate(w, template_path, data)
}
