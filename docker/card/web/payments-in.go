package web

import (
	"card/phoenix"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
)

type paymentIn struct {
	PaymentInCards      []paymentInCard
	PreviousPageEnabled string
	CurrentPageNumber   string
	NextPageEnabled     string
}

type paymentInCard struct {
	CardStyle      string
	CardHeaderText string
	CardBodyText   string
}

func PaymentsIn(w http.ResponseWriter, r *http.Request) {

	template_path := "/dist/pages/admin/payments-in/index.html"

	pmt_list, err := phoenix.ListIncomingPayments(12, 0)
	if err != nil {
		log.Warn("phoenix error: ", err.Error())
	}

	template_data := paymentIn{
		PreviousPageEnabled: "disabled",
		CurrentPageNumber:   "5",
		NextPageEnabled:     "disabled",
	}

	for _, pmt := range pmt_list {

		c := paymentInCard{
			CardStyle:      "card-warning",
			CardHeaderText: time.Unix(0, pmt.CreatedAt*int64(time.Millisecond)).Format("Mon 2 Jan 2006 15:04"),
			CardBodyText:   "",
		}

		if pmt.IsPaid {
			c.CardStyle = "card-success"
		}

		c.CardBodyText = pmt.Invoice

		template_data.PaymentInCards = append(template_data.PaymentInCards, c)
	}

	renderHtmlFromTemplate(w, template_path, template_data)
}
