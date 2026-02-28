package web

import (
	"card/db"
	"database/sql"
	"fmt"
	"net/http"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

func Admin_Index(db_conn *sql.DB, w http.ResponseWriter, r *http.Request) {

	template_path := "/admin/index.html"

	cardCount, err := db.Db_get_card_count(db_conn)
	numCards := "error"
	if err == nil {
		numCards = fmt.Sprintf("%d", cardCount)
	}

	topCards := db.Db_get_top_cards_by_balance(db_conn, 10)

	type TopCardView struct {
		CardId      string
		Note        string
		BalanceSats string
	}

	p := message.NewPrinter(language.English)

	var topCardViews []TopCardView
	for _, tc := range topCards {
		topCardViews = append(topCardViews, TopCardView{
			CardId:      fmt.Sprintf("%d", tc.CardId),
			Note:        tc.Note,
			BalanceSats: p.Sprintf("%d sats", tc.BalanceSats),
		})
	}

	data := struct {
		NumCards string
		TopCards []TopCardView
	}{
		NumCards: numCards,
		TopCards: topCardViews,
	}

	RenderHtmlFromTemplate(w, template_path, data)
}
