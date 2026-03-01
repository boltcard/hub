package web

import (
	"card/db"
	"card/phoenix"
	"net/http"

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

	incoming, err := phoenix.ListIncomingPayments(5, 0)
	if err != nil {
		log.Warn("phoenix list incoming error: ", err)
	}

	outgoing, err := phoenix.ListOutgoingPayments(5, 0)
	if err != nil {
		log.Warn("phoenix list outgoing error: ", err)
	}

	txIn := make([]txJSON, 0, len(incoming))
	for _, p := range incoming {
		if !p.IsPaid {
			continue
		}
		txIn = append(txIn, txJSON{
			Direction:   "in",
			AmountSat:   p.ReceivedSat,
			PaymentHash: p.PaymentHash,
			Timestamp:   p.CompletedAt / 1000,
			IsPaid:      p.IsPaid,
			Description: p.Description,
		})
	}

	txOut := make([]txJSON, 0, len(outgoing))
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

	writeJSON(w, map[string]interface{}{
		"in":  txIn,
		"out": txOut,
	})
}
