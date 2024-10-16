package admin

import (
	"card/build"
	"card/util"
	"card/web"
	"net/http"
)

func BoltCard(w http.ResponseWriter, r *http.Request) {

	template_path := "/dist/pages/admin/bolt-card/index.html"

	// TODO: check for a 'new bolt card' code in the database
	// TODO: create a URL for the Bolt Card Programmer app for one time bolt card creation
	createBoltCardUrl := "https://abc"

	CreateBoltCardPngEncoded := util.QrPngBase64Encode(createBoltCardUrl)

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

	web.RenderHtmlFromTemplate(w, template_path, data)
}
