package web

import (
	"card/phoenix"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
)

type paymentCard struct {
	CardStyle      string
	CardHeaderText string
	CardBodyText   string
}

func Payments(w http.ResponseWriter, r *http.Request) {

	template_path := "/dist/pages/admin/payments/index.html"

	in_pmt_list, err := phoenix.ListIncomingPayments(14, 0)
	if err != nil {
		log.Warn("phoenix error: ", err.Error())
	}

	template_data := []paymentCard{}

	for _, pmt := range in_pmt_list {
		c := paymentCard{CardStyle: "card-warning", CardHeaderText: time.Unix(0, pmt.CreatedAt*int64(time.Millisecond)).Format("2006-01-02 03:04:05") + " - Unpaid", CardBodyText: ""}

		if pmt.IsPaid {
			c.CardStyle = "card-success"
			c.CardHeaderText = time.Unix(0, pmt.CreatedAt*int64(time.Millisecond)).Format("2006-01-02 03:04:05") + " - Paid"
		}

		c.CardBodyText = pmt.Description

		template_data = append(template_data, c)
	}

	renderTemplate(w, template_path, template_data)
}
