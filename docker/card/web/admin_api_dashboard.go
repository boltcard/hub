package web

import (
	"card/db"
	"card/phoenix"
	"net/http"

	log "github.com/sirupsen/logrus"
)

func (app *App) adminApiDashboard(w http.ResponseWriter, r *http.Request) {
	cardCount, err := db.Db_get_card_count(app.db_conn)
	if err != nil {
		log.Warn("card count error: ", err)
	}

	topCards := db.Db_get_top_cards_by_balance(app.db_conn, 10)

	type topCardJSON struct {
		CardId      int    `json:"cardId"`
		Note        string `json:"note"`
		BalanceSats int    `json:"balanceSats"`
	}

	topCardViews := make([]topCardJSON, 0, len(topCards))
	for _, tc := range topCards {
		topCardViews = append(topCardViews, topCardJSON{
			CardId:      tc.CardId,
			Note:        tc.Note,
			BalanceSats: tc.BalanceSats,
		})
	}

	// Phoenix status (best effort — don't fail dashboard if Phoenix is down)
	phoenixConnected := false
	phoenixBalance := 0
	phoenixFeeCredit := 0
	balance, err := phoenix.GetBalance()
	if err == nil {
		phoenixConnected = true
		phoenixBalance = balance.BalanceSat
		phoenixFeeCredit = balance.FeeCreditSat
	}

	writeJSON(w, map[string]interface{}{
		"cardCount":        cardCount,
		"hasCards":         cardCount > 0,
		"topCards":         topCardViews,
		"phoenixConnected": phoenixConnected,
		"phoenixBalance":   phoenixBalance,
		"phoenixFeeCredit": phoenixFeeCredit,
	})
}
