package web

import (
	"card/build"
	"card/db"
	"card/util"
	"database/sql"
	"net/http"
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

func BoltCard(db_conn *sql.DB, w http.ResponseWriter, r *http.Request) {

	template_path := "/dist/pages/admin/bolt-card/index.html"

	// CreateBoltCard
	// TODO: set up a new code and update with websocket connection

	a_code := db.Db_get_setting(db_conn, "new_card_code")

	// create a URL for the Bolt Card Programmer app for bolt card creation
	createBoltCardUrl := "https://" + db.Db_get_setting(db_conn, "host_domain") + "/new?a=" + a_code
	CreateBoltCardPngEncoded := util.QrPngBase64Encode(createBoltCardUrl)

	// // WipeBoltCard
	// var wipeBoltCardJson BcpWipeResponse

	// wipeBoltCardJson.Version = 1
	// wipeBoltCardJson.Action = "wipe"
	// wipeBoltCardJson.K0 = "11111111111111111111111111111111"
	// wipeBoltCardJson.K1 = "22222222222222222222222222222222"
	// wipeBoltCardJson.K2 = "33333333333333333333333333333333"
	// wipeBoltCardJson.K3 = "44444444444444444444444444444444"
	// wipeBoltCardJson.K4 = "55555555555555555555555555555555"

	// resJson, err := json.Marshal(wipeBoltCardJson)
	// util.Check(err)

	// wipeBoltCardKeys := string(resJson)
	// //log.Info("wipeBoltCardKeys: ", wipeBoltCardKeys)
	// wipeBoltCardPngEncoded := util.QrPngBase64Encode(wipeBoltCardKeys)

	// //BatchCreateBoltCard
	// // TODO: create and replace code for batch cards
	// urlEncodedBatchCreateUrl := "https://" + db.Db_get_setting("host_domain") + "/batch/ln9n190ja1hb2owjnns1"
	// batchCreateBoltCardUrl := "boltcard://program?url=" + url.QueryEscape(urlEncodedBatchCreateUrl)

	// batchCreateBoltCardPngEncoded := util.QrPngBase64Encode(batchCreateBoltCardUrl)

	// send data to the page for rendering
	data := struct {
		CreateBoltCardPngEncoded string
		SwVersion                string
		SwBuildDate              string
		SwBuildTime              string
	}{
		CreateBoltCardPngEncoded: CreateBoltCardPngEncoded,
		SwVersion:                build.Version,
		SwBuildDate:              build.Date,
		SwBuildTime:              build.Time,
	}

	RenderHtmlFromTemplate(w, template_path, data)
}
