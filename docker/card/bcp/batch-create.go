package bcp

import (
	"card/db"
	"card/util"
	"encoding/json"
	"net/http"

	log "github.com/sirupsen/logrus"
)

type BcpBatchResponse struct {
	Lnurlw     string `json:"LNURLW"`
	Version    int    `json:"version"`
	UIDPrivacy string `json:"uid_privacy"` // looks like this is not in the BCP app yet
	K0         string `json:"K0"`
	K1         string `json:"K1"`
	K2         string `json:"K2"`
	K3         string `json:"K3"`
	K4         string `json:"K4"`
}

func BatchCreateCard(w http.ResponseWriter, r *http.Request) {
	log.Info("BatchCreateCard")

	//TODO: read authentication code from URL
	//TODO: read UID from POST vars

	var resObj BcpBatchResponse

	resObj.Version = 0
	resObj.Lnurlw = "lnurlw://" + db.Db_get_setting("host_domain") + "/ln"
	//resObj.UIDPrivacy = "N"
	resObj.K0 = "11111111111111111111111111111111"
	resObj.K1 = "22222222222222222222222222222222"
	resObj.K2 = "33333333333333333333333333333333"
	resObj.K3 = "44444444444444444444444444444444"
	resObj.K4 = "55555555555555555555555555555555"

	resJson, err := json.Marshal(resObj)
	util.Check(err)

	log.Info("resJson ", string(resJson))

	w.Write(resJson)
}
