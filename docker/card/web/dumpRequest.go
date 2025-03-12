package web

import (
	"net/http"
	"net/http/httputil"

	log "github.com/sirupsen/logrus"
)

func dumpRequest(w http.ResponseWriter, req *http.Request) {
	requestDump, err := httputil.DumpRequest(req, true)
	if err != nil {
		log.Info(err.Error())
	} else {
		log.Info(string(requestDump))
	}
}
