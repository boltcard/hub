package web

import (
	"card/db"
	"card/phoenix"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

func lnurlpMetadata(username, hostDomain string) string {
	return fmt.Sprintf(`[["text/plain","Payment to %s@%s"]]`, username, hostDomain)
}

func descriptionHash(metadata string) string {
	hash := sha256.Sum256([]byte(metadata))
	return hex.EncodeToString(hash[:])
}

func (app *App) CreateHandler_LnurlpRequest() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		vars := mux.Vars(r)
		username := vars["username"]
		if username == "" {
			w.WriteHeader(http.StatusNotFound)
			writeJSON(w, map[string]string{"status": "ERROR", "reason": "not found"})
			return
		}

		cardId := db.Db_get_card_by_ln_address(app.db_conn, username)
		if cardId == 0 {
			w.WriteHeader(http.StatusNotFound)
			writeJSON(w, map[string]string{"status": "ERROR", "reason": "not found"})
			return
		}

		hostDomain := db.Db_get_setting(app.db_conn, "host_domain")
		metadata := lnurlpMetadata(username, hostDomain)

		writeJSON(w, map[string]any{
			"tag":         "payRequest",
			"callback":    "https://" + hostDomain + "/.well-known/lnurlp/" + username + "/callback",
			"minSendable": 1000,
			"maxSendable": 100000000000,
			"metadata":    metadata,
		})
	}
}

func (app *App) CreateHandler_LnurlpCallback() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		vars := mux.Vars(r)
		username := vars["username"]
		if username == "" {
			w.WriteHeader(http.StatusNotFound)
			writeJSON(w, map[string]string{"status": "ERROR", "reason": "not found"})
			return
		}

		cardId := db.Db_get_card_by_ln_address(app.db_conn, username)
		if cardId == 0 {
			w.WriteHeader(http.StatusNotFound)
			writeJSON(w, map[string]string{"status": "ERROR", "reason": "not found"})
			return
		}

		// Validate amount (in millisats)
		amountStr := r.URL.Query().Get("amount")
		if amountStr == "" {
			w.WriteHeader(http.StatusBadRequest)
			writeJSON(w, map[string]string{"status": "ERROR", "reason": "missing amount"})
			return
		}

		amountMsat, err := strconv.ParseInt(amountStr, 10, 64)
		if err != nil || amountMsat < 1000 || amountMsat > 100000000000 {
			w.WriteHeader(http.StatusBadRequest)
			writeJSON(w, map[string]string{"status": "ERROR", "reason": "amount out of range"})
			return
		}

		amountSats := int(amountMsat / 1000)

		hostDomain := db.Db_get_setting(app.db_conn, "host_domain")
		metadata := lnurlpMetadata(username, hostDomain)
		dHash := descriptionHash(metadata)

		// Create invoice via Phoenix with description hash
		createInvoiceResponse, err := phoenix.CreateInvoice(phoenix.CreateInvoiceRequest{
			DescriptionHash: dHash,
			AmountSat:       strconv.Itoa(amountSats),
		})
		if err != nil {
			log.Error("lnurlp CreateInvoice error: ", err)
			w.WriteHeader(http.StatusInternalServerError)
			writeJSON(w, map[string]string{"status": "ERROR", "reason": "failed to create invoice"})
			return
		}

		// Insert pending receipt
		db.Db_add_card_receipt(app.db_conn, cardId,
			createInvoiceResponse.Serialized, createInvoiceResponse.PaymentHash, amountSats)

		log.Info("lnurlp invoice created for ", username, " amount=", amountSats)

		writeJSON(w, map[string]any{
			"pr":     createInvoiceResponse.Serialized,
			"routes": []string{},
		})
	}
}
