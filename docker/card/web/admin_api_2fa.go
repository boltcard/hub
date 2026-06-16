package web

import (
	"card/db"
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
