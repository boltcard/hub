package web

import (
	"card/db"
	"card/util"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

func (app *App) adminApiListCards(w http.ResponseWriter, r *http.Request) {
	cards := db.Db_select_all_cards(app.db_read)

	type cardJSON struct {
		CardId       int    `json:"cardId"`
		Uid          string `json:"uid"`
		Note         string `json:"note"`
		BalanceSats  int    `json:"balanceSats"`
		LnurlwEnable string `json:"lnurlwEnable"`
		GroupTag     string `json:"groupTag"`
		TxLimitSats  int    `json:"txLimitSats"`
		DayLimitSats int    `json:"dayLimitSats"`
	}

	result := make([]cardJSON, 0, len(cards))
	for _, c := range cards {
		result = append(result, cardJSON{
			CardId:       c.CardId,
			Uid:          c.Uid,
			Note:         c.Note,
			BalanceSats:  c.BalanceSats,
			LnurlwEnable: c.LnurlwEnable,
			GroupTag:     c.GroupTag,
			TxLimitSats:  c.TxLimitSats,
			DayLimitSats: c.DayLimitSats,
		})
	}

	writeJSON(w, map[string]any{
		"cards": result,
	})
}

// adminApiCardRouter dispatches /admin/api/cards/{id}[/action] requests.
func (app *App) adminApiCardRouter(w http.ResponseWriter, r *http.Request) {
	// Parse card ID from path: /admin/api/cards/{id}[/action]
	path := strings.TrimPrefix(r.URL.Path, "/admin/api/cards/")
	parts := strings.SplitN(path, "/", 2)

	cardId, err := strconv.Atoi(parts[0])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, map[string]string{"error": "invalid card id"})
		return
	}

	action := ""
	if len(parts) > 1 {
		action = parts[1]
	}

	switch {
	case action == "" && r.Method == "GET":
		app.adminApiGetCard(w, r, cardId)
	case action == "note" && r.Method == "PUT":
		app.adminApiUpdateCardNote(w, r, cardId)
	case action == "limits" && r.Method == "PUT":
		app.adminApiUpdateCardLimits(w, r, cardId)
	case action == "allocate" && r.Method == "POST":
		app.adminApiAllocateFunds(w, r, cardId)
	case action == "wipe" && r.Method == "POST":
		app.adminApiWipeCard(w, r, cardId)
	case action == "txs" && r.Method == "GET":
		app.adminApiCardTxs(w, r, cardId)
	default:
		w.WriteHeader(http.StatusNotFound)
		writeJSON(w, map[string]string{"error": "not found"})
	}
}

func (app *App) adminApiGetCard(w http.ResponseWriter, _ *http.Request, cardId int) {
	card, err := db.Db_get_card(app.db_read, cardId)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		writeJSON(w, map[string]string{"error": "card not found"})
		return
	}

	balance := db.Db_get_card_balance(app.db_read, cardId)

	hostDomain := db.Db_get_setting(app.db_read, "host_domain")

	writeJSON(w, map[string]any{
		"cardId":           card.Card_id,
		"uid":              card.Uid,
		"note":             card.Note,
		"balanceSats":      balance,
		"lnurlwEnable":     card.Lnurlw_enable,
		"txLimitSats":      card.Tx_limit_sats,
		"dayLimitSats":     card.Day_limit_sats,
		"pinEnable":        card.Pin_enable,
		"pinLimitSats":     card.Pin_limit_sats,
		"wiped":            card.Wiped,
		"lnAddress":        card.Ln_address,
		"lnAddressEnabled": card.Ln_address_enabled,
		"payLinkEnabled":   card.Pay_link_enabled,
		"hostDomain":       hostDomain,
	})
}

func (app *App) adminApiUpdateCardNote(w http.ResponseWriter, r *http.Request, cardId int) {
	var req struct {
		Note string `json:"note"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, map[string]string{"error": "invalid request body"})
		return
	}

	db.Db_update_card_note(app.db_write, cardId, req.Note)
	writeJSON(w, map[string]bool{"ok": true})
}

func (app *App) adminApiUpdateCardLimits(w http.ResponseWriter, r *http.Request, cardId int) {
	var req struct {
		TxLimitSats      int    `json:"txLimitSats"`
		DayLimitSats     int    `json:"dayLimitSats"`
		LnurlwEnable     string `json:"lnurlwEnable"`
		LnAddressEnabled string `json:"lnAddressEnabled"`
		PayLinkEnabled   string `json:"payLinkEnabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, map[string]string{"error": "invalid request body"})
		return
	}

	// Validate lnurlwEnable
	if req.LnurlwEnable != "Y" && req.LnurlwEnable != "N" {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, map[string]string{"error": "lnurlwEnable must be Y or N"})
		return
	}

	// Validate lnAddressEnabled (optional — default to current value)
	if req.LnAddressEnabled != "" && req.LnAddressEnabled != "Y" && req.LnAddressEnabled != "N" {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, map[string]string{"error": "lnAddressEnabled must be Y or N"})
		return
	}

	// Use the update without pin variant — admin doesn't change PIN settings
	db.Db_update_card_without_pin(app.db_write, cardId, req.TxLimitSats,
		req.DayLimitSats, "N", 0, req.LnurlwEnable)

	if req.LnAddressEnabled != "" {
		db.Db_update_card_ln_address_enabled(app.db_write, cardId, req.LnAddressEnabled)
	}

	// Validate payLinkEnabled (optional)
	if req.PayLinkEnabled != "" && req.PayLinkEnabled != "Y" && req.PayLinkEnabled != "N" {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, map[string]string{"error": "payLinkEnabled must be Y or N"})
		return
	}

	if req.PayLinkEnabled != "" {
		db.Db_update_card_pay_link_enabled(app.db_write, cardId, req.PayLinkEnabled)
	}

	writeJSON(w, map[string]bool{"ok": true})
}

// adminApiAllocateFunds credits a card with a manual balance top-up. It records
// a paid card_receipt (mirroring the SetupCardAmountForTag CLI command) so the
// allocation shows up in the card's transaction history and balance.
func (app *App) adminApiAllocateFunds(w http.ResponseWriter, r *http.Request, cardId int) {
	var req struct {
		AmountSats int `json:"amountSats"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, map[string]string{"error": "invalid request body"})
		return
	}

	// card_receipts has a CHECK (amount_sats > 0) constraint
	if req.AmountSats <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, map[string]string{"error": "amountSats must be greater than 0"})
		return
	}

	// Db_get_card filters wiped='N', so this also rejects wiped/unknown cards
	if _, err := db.Db_get_card(app.db_read, cardId); err != nil {
		w.WriteHeader(http.StatusNotFound)
		writeJSON(w, map[string]string{"error": "card not found"})
		return
	}

	// a unique r_hash_hex is required (UNIQUE constraint); use a random value
	// since there is no real Lightning invoice behind a manual allocation
	receiptId := db.Db_add_card_receipt(app.db_write, cardId, "", util.Random_hex(), req.AmountSats)
	if receiptId == 0 {
		w.WriteHeader(http.StatusInternalServerError)
		writeJSON(w, map[string]string{"error": "failed to allocate funds"})
		return
	}
	db.Db_update_receipt_paid(app.db_write, receiptId)

	balance := db.Db_get_card_balance(app.db_read, cardId)

	log.Info("admin allocated funds: card=", cardId, " amount=", req.AmountSats)
	writeJSON(w, map[string]any{
		"ok":          true,
		"balanceSats": balance,
	})
}

func (app *App) adminApiWipeCard(w http.ResponseWriter, _ *http.Request, cardId int) {
	keys := db.Db_wipe_card(app.db_write, cardId)
	if keys.Key0 == "" {
		w.WriteHeader(http.StatusNotFound)
		writeJSON(w, map[string]string{"error": "card not found"})
		return
	}

	// Issue a short-lived capability secret and build a Bolt Card programmer
	// deeplink to /wipe?s=<secret>. The app fetches that URL to get the card's
	// keys and reset the physical NFC chip. The card is already disabled above.
	// Per the Bolt Card DEEPLINK.md spec the wipe uses the "reset" action (not
	// "program"), so the app runs its reset flow: authenticate with the card's
	// current keys and restore them to the factory defaults.
	secret := util.Random_hex()
	expiry := time.Now().Unix() + 24*60*60 // 24h to complete the reset
	db.Db_set_card_wipe_secret(app.db_write, cardId, secret, expiry)

	hostDomain := db.Db_get_setting(app.db_read, "host_domain")
	wipeUrl := "https://" + hostDomain + "/wipe?s=" + secret
	boltcardLink := "boltcard://reset?url=" + url.QueryEscape(wipeUrl)

	log.Info("admin wiped card: ", cardId)
	writeJSON(w, map[string]any{
		"ok":           true,
		"boltcardLink": boltcardLink,
		"wipeUrl":      wipeUrl,
	})
}

func (app *App) adminApiCardTxs(w http.ResponseWriter, _ *http.Request, cardId int) {
	txs := db.Db_select_card_txs(app.db_read, cardId)

	type txJSON struct {
		ReceiptId  int  `json:"receiptId"`
		PaymentId  int  `json:"paymentId"`
		Timestamp  int  `json:"timestamp"`
		AmountSats int  `json:"amountSats"`
		FeeSats    int  `json:"feeSats"`
		Allocated  bool `json:"allocated"`
	}

	result := make([]txJSON, 0, len(txs))
	for _, tx := range txs {
		result = append(result, txJSON{
			ReceiptId:  tx.ReceiptId,
			PaymentId:  tx.PaymentId,
			Timestamp:  tx.Timestamp,
			AmountSats: tx.AmountSats,
			FeeSats:    tx.FeeSats,
			Allocated:  tx.Allocated,
		})
	}

	writeJSON(w, map[string]any{
		"txs": result,
	})
}

func (app *App) adminApiBatchCreate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		GroupTag       string `json:"groupTag"`
		MaxCards       int    `json:"maxCards"`
		InitialBalance int    `json:"initialBalance"`
		ExpiryHours    int    `json:"expiryHours"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, map[string]string{"error": "invalid request body"})
		return
	}

	if req.MaxCards <= 0 || req.ExpiryHours <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, map[string]string{"error": "maxCards and expiryHours are required"})
		return
	}

	secret := util.Random_hex()
	createTime := int(time.Now().Unix())
	expireTime := createTime + req.ExpiryHours*60*60

	db.Db_insert_program_cards(app.db_write, secret, req.GroupTag,
		req.MaxCards, req.InitialBalance, createTime, expireTime)

	hostDomain := db.Db_get_setting(app.db_read, "host_domain")
	programUrl := "https://" + hostDomain + "/batch?s=" + secret
	boltcardLink := "boltcard://program?url=" + url.QueryEscape(programUrl)

	qrBase64 := util.QrPngBase64Encode(boltcardLink)

	log.Info("admin created batch: group=", req.GroupTag, " max=", req.MaxCards)
	writeJSON(w, map[string]any{
		"ok":           true,
		"boltcardLink": boltcardLink,
		"programUrl":   programUrl,
		"qr":           qrBase64,
	})
}
