package web

import (
	"card/phoenix"
	"encoding/json"
	"net/http"

	log "github.com/sirupsen/logrus"
)

// adminApiPhoenixSeed returns the phoenixd wallet recovery phrase (BIP39
// mnemonic). This is the highest-value secret in the system — anyone holding
// it controls the wallet — so beyond the admin session it requires the admin
// to re-enter their password in the request body. Every attempt is logged for
// audit. The endpoint is POST-only so the secret is never placed in a URL,
// and the dispatcher already sets Cache-Control: no-store on the response.
func (app *App) adminApiPhoenixSeed(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, map[string]string{"error": "invalid request body"})
		return
	}

	if !app.verifyAdminPassword(req.Password) {
		log.Warn("phoenix seed reveal denied: invalid password")
		w.WriteHeader(http.StatusUnauthorized)
		writeJSON(w, map[string]string{"error": "invalid password"})
		return
	}

	words, err := phoenix.GetSeedWords()
	if err != nil {
		log.Warn("phoenix seed reveal failed: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		writeJSON(w, map[string]string{"error": "seed not available"})
		return
	}

	log.Warn("phoenix seed revealed to authenticated admin")
	writeJSON(w, map[string]interface{}{"words": words})
}
