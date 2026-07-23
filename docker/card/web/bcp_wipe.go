package web

import (
	"card/db"
	"net/http"

	log "github.com/sirupsen/logrus"
)

// BcpWipeResponse is what the Bolt Card app expects when resetting a card: the
// card's current keys. The app accepts either K0-K4 or k0-k4; we send K0-K4 to
// match the /batch programming response.
type BcpWipeResponse struct {
	K0 string `json:"K0"`
	K1 string `json:"K1"`
	K2 string `json:"K2"`
	K3 string `json:"K3"`
	K4 string `json:"K4"`
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

		writeJSON(w, BcpWipeResponse{
			K0: keys.Key0,
			K1: keys.Key1,
			K2: keys.Key2,
			K3: keys.Key3,
			K4: keys.Key4,
		})
	}
}
