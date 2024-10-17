package lnurlw

import (
	"card/crypto"
	"card/db"
	"card/util"

	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"
)

func get_p_c(req *http.Request) (p []byte, c []byte, err error) {

	p = []byte{}
	c = []byte{}

	params_p, ok := req.URL.Query()["p"]

	if !ok || len(params_p[0]) < 1 {
		return p, c, errors.New("p value not found")
	}

	params_c, ok := req.URL.Query()["c"]

	if !ok || len(params_c[0]) < 1 {
		return p, c, errors.New("c value not found")
	}

	p_str := params_p[0]
	c_str := params_c[0]

	p, err = hex.DecodeString(p_str)

	if err != nil {
		return p, c, errors.New("p parameter not valid hex")
	}

	c, err = hex.DecodeString(c_str)

	if err != nil {
		return p, c, errors.New("c parameter not valid hex")
	}

	if len(p) != 16 {
		return p, c, errors.New("p parameter length not valid")
	}

	if len(c) != 8 {
		return p, c, errors.New("c parameter length not valid")
	}

	return p, c, nil
}

func check_cmac(uid []byte, ctr []byte, key2_cmac []byte, cmac []byte) (bool, error) {

	sv2 := make([]byte, 16)
	sv2[0] = 0x3c
	sv2[1] = 0xc3
	sv2[2] = 0x00
	sv2[3] = 0x01
	sv2[4] = 0x00
	sv2[5] = 0x80
	sv2[6] = uid[0]
	sv2[7] = uid[1]
	sv2[8] = uid[2]
	sv2[9] = uid[3]
	sv2[10] = uid[4]
	sv2[11] = uid[5]
	sv2[12] = uid[6]
	sv2[13] = ctr[0]
	sv2[14] = ctr[1]
	sv2[15] = ctr[2]

	cmac_verified, err := crypto.Aes_cmac(key2_cmac, sv2, cmac)

	if err != nil {
		return false, err
	}

	return cmac_verified, nil
}

func check_card_tap(p []byte, c []byte, key1_str string, k2_str string) (card_found bool, uid_str string, counter uint32) {

	key_sdm_file_read, err := hex.DecodeString(key1_str)

	if err != nil {
		return false, "", 0
	}

	dec_p, err := crypto.Aes_decrypt(key_sdm_file_read, p)

	if err != nil {
		return false, "", 0
	}

	if dec_p[0] != 0xC7 {
		return false, "", 0
	}

	decoded_uid := dec_p[1:8]
	decoded_ctr := dec_p[8:11]

	key2_cmac, err := hex.DecodeString(k2_str)

	if err != nil {
		return false, "", 0
	}

	cmac_valid, err := check_cmac(decoded_uid, decoded_ctr, key2_cmac, c)

	if err != nil {
		return false, "", 0
	}

	if !cmac_valid {
		return false, "", 0
	}

	uid_str = hex.EncodeToString(decoded_uid)
	counter = uint32(decoded_ctr[2])<<16 | uint32(decoded_ctr[1])<<8 | uint32(decoded_ctr[0])

	return true, uid_str, counter
}

type LnurlwResponse struct {
	Tag                 string `json:"tag"`
	Callback            string `json:"callback"`
	Lnurlwk1            string `json:"k1"`
	DefaultDescrription string `json:"default_description"`
	MinWithdrawable     int    `json:"minWithdrawable"`
	MaxWithdrawable     int    `json:"maxWithdrawable"`
	PinLimit            int    `json:"pinLimit,omitempty"`
}

func LnurlwRequest(w http.ResponseWriter, req *http.Request) {

	log.Info("LnurlwRequest received")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	p, c, err := get_p_c(req)
	if err != nil {
		w.Write([]byte(`{"status": "ERROR", "reason": "badly formatted request"}`))
		return
	}

	cardKeys := db.Db_get_card_keys()

	log.Info("number of cards to check is ", len(cardKeys))

	cardMatch := false
	var cardId int
	var ctr uint32

	for _, cardKey := range cardKeys {

		card_match, _, match_ctr := check_card_tap(p, c, cardKey.Key1, cardKey.Key2)

		if card_match {
			cardMatch = true
			cardId = cardKey.CardId
			ctr = match_ctr
			break
		}
	}

	if !cardMatch {
		log.Info("card not found")
		w.Write([]byte(`{"status": "ERROR", "reason": "card not found"}`))
		return
	}

	log.Info("card found")

	// check counter is incremented
	cardLastCounter := db.Db_get_card_counter(cardId)
	if ctr <= cardLastCounter {
		log.Info("card counter not incremented")
		w.Write([]byte(`{"status": "ERROR", "reason": "card counter not incremented"}`))
		return
	}

	// store new counter value
	db.Db_set_card_counter(cardId, ctr)

	// create and store lnurlw_k1
	lnurlwK1 := util.Random_hex()
	lnurlwK1Expiry := time.Now().Unix() + 10 // TODO: get timeout setting
	db.Db_set_lnurlw_k1(cardId, lnurlwK1, lnurlwK1Expiry)

	// prepare response
	var resObj LnurlwResponse

	resObj.Tag = "withdrawRequest"
	resObj.Callback = "https://" + db.Db_get_setting("host_domain") + "/cb"
	resObj.Lnurlwk1 = lnurlwK1
	resObj.MinWithdrawable, _ = strconv.Atoi(db.Db_get_setting("min_withdraw_sats"))
	resObj.MaxWithdrawable, _ = strconv.Atoi(db.Db_get_setting("max_withdraw_sats"))

	log.Info("card response sent")

	// send response
	resJson, err := json.Marshal(resObj)
	util.Check(err)

	w.Write(resJson)
}
