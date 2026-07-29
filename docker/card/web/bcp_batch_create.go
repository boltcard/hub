package web

import (
	"card/db"
	"card/util"
	"encoding/json"
	"errors"
	"net/http"

	log "github.com/sirupsen/logrus"
)

// Bolt Card Programmer app section that uses this data
// https://github.com/boltcard/bolt-nfc-android-app/blob/ad83c2bf9fe631df38111a272bba2d9ea8e0df4e/src/components/SetupBoltcard.js#L128
type BcpBatchResponse struct {
	Lnurlw     string `json:"LNURLW"`
	K0         string `json:"K0"`
	K1         string `json:"K1"`
	K2         string `json:"K2"`
	K3         string `json:"K3"`
	K4         string `json:"K4"`
	UIDPrivacy string `json:"uid_privacy"` // looks like this is not in the BCP app yet
}

// bcpError replies with the Bolt Card Programmer error shape used by /new
// ({"status": "ERROR", "reason": ...}) alongside an HTTP error status, so a
// client can tell an exhausted batch from an expired one.
func bcpError(w http.ResponseWriter, status int, reason string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(map[string]string{
		"status": "ERROR",
		"reason": reason,
	}); err != nil {
		log.Error("bcpError encode error: ", err)
	}
}

func (app *App) CreateHandler_BatchCreateCard() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		log.Info("BatchCreateCard")

		// POST /batch?s=46f7e693a32efea0a505e39db464dd6f
		// {"UID":"048B71B22D6B80"}

		// get secret param
		secret := r.URL.Query().Get("s")

		// get UID param
		decoder := json.NewDecoder(r.Body)
		t := struct {
			Uid string `json:"UID"`
		}{}

		err := decoder.Decode(&t)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		log.Info("Uid : ", t.Uid)

		// Claim a slot in the batch and create the card in one transaction:
		// the secret has to be live, within its expiry window, and still have
		// cards left on its max_group_num allowance. Each POST here mints a
		// real card, so an unbounded batch link is a card printing press.
		k0, k1, k2, k3, k4 := generateCardKeys()
		keys := db.CardKeys{Key0: k0, Key1: k1, Key2: k2, Key3: k3, Key4: k4}
		login := util.Random_hex()
		password := util.Random_hex()

		cardId, _, err := db.Db_dispense_batch_card(app.db_write, secret, t.Uid, keys, login, password)
		if err != nil {
			switch {
			case errors.Is(err, db.ErrBatchExhausted):
				log.Warn("batch has already dispensed all of its cards")
				bcpError(w, http.StatusBadRequest, "batch limit reached")
			case errors.Is(err, db.ErrBatchNotFound), errors.Is(err, db.ErrBatchExpired):
				log.Warn("ProgramCard record within expiry time not found")
				bcpError(w, http.StatusBadRequest, "program card expired or not found")
			default:
				log.Error("batch card dispense error: ", err)
				bcpError(w, http.StatusInternalServerError, "could not create card")
			}
			return
		}

		log.Info("batch dispensed card_id ", cardId)

		var bcpBatchResponse BcpBatchResponse

		bcpBatchResponse.Lnurlw = "lnurlw://" + db.Db_get_setting(app.db_read, "host_domain") + "/ln"
		bcpBatchResponse.UIDPrivacy = "Y"
		bcpBatchResponse.K0 = k0
		bcpBatchResponse.K1 = k1
		bcpBatchResponse.K2 = k2
		bcpBatchResponse.K3 = k3
		bcpBatchResponse.K4 = k4

		writeJSON(w, bcpBatchResponse)
	}
}
