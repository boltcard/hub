package web

import (
	"card/db"
	"card/util"
	"encoding/json"
	"net/http"

	log "github.com/sirupsen/logrus"
)

// totpEnabled reports whether admin TOTP 2FA is currently active.
func (app *App) totpEnabled() bool {
	return db.Db_get_setting(app.db_read, "admin_totp_enabled") == "Y"
}

// loadRecoveryHashes returns the stored bcrypt hashes of unused recovery codes.
func (app *App) loadRecoveryHashes() []string {
	raw := db.Db_get_setting(app.db_read, "admin_totp_recovery_hash")
	if raw == "" {
		return nil
	}
	var hashes []string
	if err := json.Unmarshal([]byte(raw), &hashes); err != nil {
		log.Error("recovery hash unmarshal error: ", err)
		return nil
	}
	return hashes
}

// saveRecoveryHashes persists the recovery-code hashes as a JSON array.
func (app *App) saveRecoveryHashes(hashes []string) {
	b, err := json.Marshal(hashes)
	if err != nil {
		log.Error("recovery hash marshal error: ", err)
		return
	}
	db.Db_set_setting(app.db_write, "admin_totp_recovery_hash", string(b))
}

// consumeRecoveryCode returns true if code matches an unused recovery code,
// removing it (single use) before returning.
func (app *App) consumeRecoveryCode(code string) bool {
	hashes := app.loadRecoveryHashes()
	idx, ok := matchRecoveryCode(code, hashes)
	if !ok {
		return false
	}
	hashes = append(hashes[:idx], hashes[idx+1:]...)
	app.saveRecoveryHashes(hashes)
	return true
}

func (app *App) adminApi2faStatus(w http.ResponseWriter, r *http.Request) {
	enabled := app.totpEnabled()
	remaining := 0
	if enabled {
		remaining = len(app.loadRecoveryHashes())
	}
	writeJSON(w, map[string]interface{}{
		"enabled":                enabled,
		"recoveryCodesRemaining": remaining,
	})
}

func (app *App) adminApi2faSetup(w http.ResponseWriter, r *http.Request) {
	if app.totpEnabled() {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, map[string]string{"error": "2fa already enabled"})
		return
	}

	hostDomain := db.Db_get_setting(app.db_read, "host_domain")
	secret, url, err := newTotpKey(hostDomain)
	if err != nil {
		log.Error("totp generate error: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		writeJSON(w, map[string]string{"error": "internal error"})
		return
	}

	// Store the pending secret; it is inert until enable sets enabled=Y.
	if db.Db_get_setting(app.db_read, "admin_totp_secret") != "" {
		log.Info("replacing pending (unenabled) TOTP secret")
	}
	db.Db_set_setting(app.db_write, "admin_totp_secret", secret)
	db.Db_set_setting(app.db_write, "admin_totp_enabled", "N")

	writeJSON(w, map[string]string{
		"secret":     secret,
		"otpauthUri": url,
		"qrPng":      util.QrPngBase64Encode(url),
	})
}

func (app *App) adminApi2faEnable(w http.ResponseWriter, r *http.Request) {
	if app.totpEnabled() {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, map[string]string{"error": "2fa already enabled"})
		return
	}

	var req struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, map[string]string{"error": "invalid request body"})
		return
	}

	secret := db.Db_get_setting(app.db_read, "admin_totp_secret")
	if secret == "" {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, map[string]string{"error": "no pending 2fa setup"})
		return
	}
	if !validateTotpCode(secret, req.Code) {
		// session is valid; only the submitted code is wrong → 400, not 401,
		// so the shared apiFetch does not mislabel it as "session expired".
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, map[string]string{"error": "invalid code"})
		return
	}

	plain, hashes, err := generateRecoveryCodes(10)
	if err != nil {
		log.Error("recovery code generation error: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		writeJSON(w, map[string]string{"error": "internal error"})
		return
	}
	app.saveRecoveryHashes(hashes)
	db.Db_set_setting(app.db_write, "admin_totp_enabled", "Y")

	writeJSON(w, map[string]interface{}{"recoveryCodes": plain})
}
