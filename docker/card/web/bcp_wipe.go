package web

import (
	"card/db"
	"net/http"

	log "github.com/sirupsen/logrus"
)

// BcpWipeResponse is what the Bolt Card app expects when resetting a card. The
// app's "Check URLs and Keys" screen (bolt-card-programmer DisplayAuthInfo.tsx)
// requires an LNURLW/lnurlw_base field alongside the keys, or it rejects the
// response ("The JSON response must contain lnurlw_base, k0, k1, k2, k3, k4").
// The app accepts either K0-K4 or k0-k4; we send K0-K4 to match /batch.
type BcpWipeResponse struct {
	LNURLW string `json:"LNURLW"`
	K0     string `json:"K0"`
	K1     string `json:"K1"`
	K2     string `json:"K2"`
	K3     string `json:"K3"`
	K4     string `json:"K4"`
}

// CreateHandler_WipeCard serves a card's keys for a valid wipe capability
// secret so the Bolt Card app can reset the physical NFC chip. It is reached
// via the boltcard://program?url=<host>/wipe?s=<secret> deeplink produced by
// the admin "Wipe Card" action; the app POSTs the card's current NDEF and
// expects the keys back.
func (app *App) CreateHandler_WipeCard() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		log.Info("WipeCard request received")

		secret := r.URL.Query().Get("s")
		keys := db.Db_get_card_keys_for_wipe_secret(app.db_read, secret)
		if keys.Key0 == "" {
			log.Info("wipe secret not found or expired")
			http.Error(w, "wipe link expired or not found", http.StatusBadRequest)
			return
		}

		hostDomain := db.Db_get_setting(app.db_read, "host_domain")
		writeJSON(w, BcpWipeResponse{
			LNURLW: "lnurlw://" + hostDomain + "/ln",
			K0:     keys.Key0,
			K1:     keys.Key1,
			K2:     keys.Key2,
			K3:     keys.Key3,
			K4:     keys.Key4,
		})
	}
}
