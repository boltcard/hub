package web

import (
	"card/db"
	"card/phoenix"
	"net/http"
	"sort"

	log "github.com/sirupsen/logrus"
)

func (app *App) adminApiTransactions(w http.ResponseWriter, r *http.Request) {
	type txJSON struct {
		Direction   string `json:"direction"`
		AmountSat   int    `json:"amountSat"`
		PaymentHash string `json:"paymentHash"`
		Timestamp   int64  `json:"timestamp"`
		IsPaid      bool   `json:"isPaid"`
		Description string `json:"description,omitempty"`
		CardNote    string `json:"cardNote,omitempty"`
	}

	const fetchLimit = 500 // Phoenix returns oldest-first; fetch all to find the most recent
	const displayLimit = 5

	incoming, err := phoenix.ListIncomingPayments(fetchLimit, 0)
	if err != nil {
		log.Warn("phoenix list incoming error: ", err)
	}

	outgoing, err := phoenix.ListOutgoingPayments(fetchLimit, 0)
	if err != nil {
		log.Warn("phoenix list outgoing error: ", err)
	}

	txIn := make([]txJSON, 0, displayLimit)
	for _, p := range incoming {
		if !p.IsPaid {
			continue
		}
		message := p.PayerNote
		if message == "" {
			message = p.Description
		}
		txIn = append(txIn, txJSON{
			Direction:   "in",
			AmountSat:   p.ReceivedSat,
			PaymentHash: p.PaymentHash,
			Timestamp:   p.CompletedAt / 1000,
			IsPaid:      p.IsPaid,
			Description: message,
		})
	}

	txOut := make([]txJSON, 0, displayLimit)
	for _, p := range outgoing {
		if !p.IsPaid {
			continue
		}
		cardNote := db.Db_get_card_note_by_invoice(app.db_conn, p.Invoice)
		txOut = append(txOut, txJSON{
			Direction:   "out",
			AmountSat:   p.Sent,
			PaymentHash: p.PaymentHash,
			Timestamp:   p.CompletedAt / 1000,
			IsPaid:      p.IsPaid,
			CardNote:    cardNote,
		})
	}

	sort.Slice(txIn, func(i, j int) bool { return txIn[i].Timestamp > txIn[j].Timestamp })
	sort.Slice(txOut, func(i, j int) bool { return txOut[i].Timestamp > txOut[j].Timestamp })

	if len(txIn) > displayLimit {
		txIn = txIn[:displayLimit]
	}
	if len(txOut) > displayLimit {
		txOut = txOut[:displayLimit]
	}

	writeJSON(w, map[string]interface{}{
		"in":  txIn,
		"out": txOut,
	})
}
