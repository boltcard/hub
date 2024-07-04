package web

import (
	"card/phoenix"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
)

type paymentOut struct {
	PaymentOutCards     []paymentOutCard
	PreviousPageEnabled string
	CurrentPageNumber   string
	NextPageEnabled     string
}

type paymentOutCard struct {
	CardStyle      string
	CardHeaderText string
	CardBodyText   string
}

func PaymentsOut(w http.ResponseWriter, r *http.Request) {

	template_path := "/dist/pages/admin/payments-out/index.html"

	pmt_list, err := phoenix.ListOutgoingPayments(12, 0)
	if err != nil {
		log.Warn("phoenix error: ", err.Error())
	}

	template_data := paymentOut{
		PreviousPageEnabled: "disabled",
		CurrentPageNumber:   "5",
		NextPageEnabled:     "disabled",
	}

	for _, pmt := range pmt_list {

		c := paymentOutCard{
			CardStyle:      "card-warning",
			CardHeaderText: time.Unix(0, pmt.CreatedAt*int64(time.Millisecond)).Format("Mon 2 Jan 2006 15:04"),
			CardBodyText:   "",
		}

		if pmt.IsPaid {
			c.CardStyle = "card-success"
		}

		c.CardBodyText = pmt.Invoice

		template_data.PaymentOutCards = append(template_data.PaymentOutCards, c)
	}

	renderHtmlFromTemplate(w, template_path, template_data)
}
