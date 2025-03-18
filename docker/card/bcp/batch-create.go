package bcp

import (
	"card/db"
	"card/util"
	"encoding/json"
	"net/http"
	"time"

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

func BatchCreateCard(w http.ResponseWriter, r *http.Request) {
	log.Info("BatchCreateCard")

	// POST /batch?s=46f7e693a32efea0a505e39db464dd6f
	// {"UID":"048B71B22D6B80"}

	// get secret param
	secret := r.URL.Query().Get("s")

	log.Info("secret : ", secret)

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

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// check secret in program_cards exists and is not expired, and get group_tag field
	programCard := db.Db_select_program_card_for_secret(secret)
	currentTime := int(time.Now().Unix())

	if currentTime < programCard.CreateTime || currentTime > programCard.ExpireTime {
		log.Warn("ProgramCard record within expiry time not found")
		return
	}

	// create a new card in the database
	k0 := random_hex()
	k1 := random_hex()
	k2 := random_hex()
	k3 := random_hex()
	k4 := random_hex()
	login := random_hex() // included for LndHub compatibility
	password := random_hex()
	db.Db_insert_card_with_uid(k0, k1, k2, k3, k4, login, password, t.Uid, programCard.GroupTag)

	var bcpBatchResponse BcpBatchResponse

	bcpBatchResponse.Lnurlw = "lnurlw://" + db.Db_get_setting("host_domain") + "/ln"
	bcpBatchResponse.UIDPrivacy = "Y"
	bcpBatchResponse.K0 = k0
	bcpBatchResponse.K1 = k1
	bcpBatchResponse.K2 = k2
	bcpBatchResponse.K3 = k3
	bcpBatchResponse.K4 = k4

	resJson, err := json.Marshal(bcpBatchResponse)
	util.CheckAndPanic(err)

	log.Info("resJson ", string(resJson))

	w.Write(resJson)
}
