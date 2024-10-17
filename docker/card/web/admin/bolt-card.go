package admin

import (
	"card/build"
	"card/db"
	"card/util"
	"card/web"
	"encoding/json"
	"net/http"
	"net/url"
)

type BcpWipeResponse struct {
	Version int    `json:"version"`
	Action  string `json:"action"`
	K0      string `json:"k0"`
	K1      string `json:"k1"`
	K2      string `json:"k2"`
	K3      string `json:"k3"`
	K4      string `json:"k4"`
}

func BoltCard(w http.ResponseWriter, r *http.Request) {

	template_path := "/dist/pages/admin/bolt-card/index.html"

	// CreateBoltCard
	// TODO: check for a 'new bolt card' code in the database
	a_code := "00"
	// TODO: create a URL for the Bolt Card Programmer app for one time bolt card creation
	createBoltCardUrl := "https://" + db.Db_get_setting("host_domain") + "/new?a=" + a_code

	CreateBoltCardPngEncoded := util.QrPngBase64Encode(createBoltCardUrl)

	// WipeBoltCard
	var wipeBoltCardJson BcpWipeResponse

	wipeBoltCardJson.Version = 1
	wipeBoltCardJson.Action = "wipe"
	wipeBoltCardJson.K0 = "11111111111111111111111111111111"
	wipeBoltCardJson.K1 = "22222222222222222222222222222222"
	wipeBoltCardJson.K2 = "33333333333333333333333333333333"
	wipeBoltCardJson.K3 = "44444444444444444444444444444444"
	wipeBoltCardJson.K4 = "55555555555555555555555555555555"

	resJson, err := json.Marshal(wipeBoltCardJson)
	util.Check(err)

	wipeBoltCardKeys := string(resJson)
	//log.Info("wipeBoltCardKeys: ", wipeBoltCardKeys)
	wipeBoltCardPngEncoded := util.QrPngBase64Encode(wipeBoltCardKeys)

	//BatchCreateBoltCard
	urlEncodedBatchCreateUrl := "https://" + db.Db_get_setting("host_domain") + "/batch/ln9n190ja1hb2owjnns1"
	batchCreateBoltCardUrl := "boltcard://program?url=" + url.QueryEscape(urlEncodedBatchCreateUrl)

	batchCreateBoltCardPngEncoded := util.QrPngBase64Encode(batchCreateBoltCardUrl)

	// send data to the page for rendering
	data := struct {
		CreateBoltCardPngEncoded      string
		WipeBoltCardPngEncoded        string
		BatchCreateBoltCardPngEncoded string
		SwVersion                     string
		SwBuildDate                   string
		SwBuildTime                   string
	}{
		CreateBoltCardPngEncoded:      CreateBoltCardPngEncoded,
		WipeBoltCardPngEncoded:        wipeBoltCardPngEncoded,
		BatchCreateBoltCardPngEncoded: batchCreateBoltCardPngEncoded,
		SwVersion:                     build.Version,
		SwBuildDate:                   build.Date,
		SwBuildTime:                   build.Time,
	}

	web.RenderHtmlFromTemplate(w, template_path, data)
}
