package web

import (
	"card/db"
	"card/phoenix"
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

// verifyAdminPassword checks a password against the stored admin hash
// (bcrypt or legacy SHA256). Verification only — it does not migrate.
func (app *App) verifyAdminPassword(password string) bool {
	hash := db.Db_get_setting(app.db_read, "admin_password_hash")
	if hash == "" {
		return false
	}
	if isBcryptHash(hash) {
		return CheckPassword(password, hash)
	}
	legacyHash := GetPwHash(app.db_read, password)
	return subtle.ConstantTimeCompare([]byte(legacyHash), []byte(hash)) == 1
}

type withdrawalJSON struct {
	AmountSats  int    `json:"amountSats"`
	FeeSats     int    `json:"feeSats"`
	LnAddress   string `json:"lnAddress"`
	PaymentHash string `json:"paymentHash"`
	Status      string `json:"status"`
	Timestamp   int    `json:"timestamp"`
}

// adminApiWithdrawInfo reports how much the admin can safely withdraw and the
// recent withdrawal history. The "excess" is node balance minus the total
// outstanding card balance (the liability owed to cardholders).
func (app *App) adminApiWithdrawInfo(w http.ResponseWriter, r *http.Request) {
	nodeBalanceSat := 0
	balance, err := phoenix.GetBalance()
	if err != nil {
		// Best effort — node may be down. Avoid logging the error value
		// itself: it can carry response bytes from a credentialed request.
		log.Warn("withdraw info: phoenix balance unavailable")
	} else {
		nodeBalanceSat = balance.BalanceSat
	}

	cardLiabilitySat := db.Db_get_total_card_balance(app.db_read)
	excessSat := nodeBalanceSat - cardLiabilitySat
	if excessSat < 0 {
		excessSat = 0
	}

	recent := db.Db_select_admin_withdrawals(app.db_read, 10)
	views := make([]withdrawalJSON, 0, len(recent))
	for _, wd := range recent {
		views = append(views, withdrawalJSON{
			AmountSats:  wd.AmountSats,
			FeeSats:     wd.FeeSats,
			LnAddress:   wd.LnAddress,
			PaymentHash: wd.PaymentHash,
			Status:      wd.Status,
			Timestamp:   wd.Timestamp,
		})
	}

	writeJSON(w, map[string]interface{}{
		"nodeBalanceSat":   nodeBalanceSat,
		"cardLiabilitySat": cardLiabilitySat,
		"excessSat":        excessSat,
		"recent":           views,
	})
}

// adminApiWithdraw pays out node liquidity to a Lightning address. It requires
// the admin password to be re-entered and caps the amount at the node balance.
// Withdrawing below the card liability is allowed but flagged to the caller.
func (app *App) adminApiWithdraw(w http.ResponseWriter, r *http.Request) {
	var req struct {
		LnAddress string `json:"lnAddress"`
		AmountSat int    `json:"amountSat"`
		Message   string `json:"message"`
		Password  string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, map[string]string{"error": "invalid request body"})
		return
	}

	req.LnAddress = strings.TrimSpace(req.LnAddress)

	// Re-confirm the admin password on top of the session cookie.
	if !app.verifyAdminPassword(req.Password) {
		w.WriteHeader(http.StatusUnauthorized)
		writeJSON(w, map[string]string{"error": "invalid password"})
		return
	}

	// Basic validation of the destination Lightning address.
	at := strings.Index(req.LnAddress, "@")
	if at <= 0 || at == len(req.LnAddress)-1 || strings.Count(req.LnAddress, "@") != 1 {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, map[string]string{"error": "invalid lightning address"})
		return
	}

	if req.AmountSat <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, map[string]string{"error": "amount must be greater than zero"})
		return
	}

	// Enforce the hard cap: cannot pay more than the node holds.
	balance, err := phoenix.GetBalance()
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		writeJSON(w, map[string]string{"error": "could not read node balance"})
		return
	}
	if req.AmountSat > balance.BalanceSat {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, map[string]string{"error": "amount exceeds node balance"})
		return
	}

	cardLiabilitySat := db.Db_get_total_card_balance(app.db_read)
	breachesLiability := balance.BalanceSat-req.AmountSat < cardLiabilitySat

	// Record the attempt before paying so a timeout still leaves a trail.
	withdrawalId, err := db.Db_insert_admin_withdrawal(app.db_write, req.LnAddress, req.AmountSat)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		writeJSON(w, map[string]string{"error": "could not record withdrawal"})
		return
	}

	payResponse, reason, err := phoenix.PayLightningAddress(phoenix.PayLightningAddressRequest{
		AmountSat: strconv.Itoa(req.AmountSat),
		Address:   req.LnAddress,
		Message:   req.Message,
	})
	if err != nil || reason != "no_error" {
		db.Db_update_admin_withdrawal_failed(app.db_write, withdrawalId)
		log.Warn("admin withdrawal failed: reason=", reason, " err=", err)
		w.WriteHeader(http.StatusBadGateway)
		writeJSON(w, map[string]string{"error": "payment failed: " + reason})
		return
	}

	db.Db_update_admin_withdrawal_paid(app.db_write, withdrawalId,
		payResponse.RoutingFeeSat, payResponse.PaymentHash)

	log.Info("admin withdrawal paid: amount=", req.AmountSat, " to=", req.LnAddress,
		" fee=", payResponse.RoutingFeeSat)

	app.broadcastPaymentSent(req.AmountSat, payResponse.PaymentHash, time.Now().Unix())

	writeJSON(w, map[string]interface{}{
		"ok":                true,
		"paymentHash":       payResponse.PaymentHash,
		"feeSat":            payResponse.RoutingFeeSat,
		"breachesLiability": breachesLiability,
	})
}
