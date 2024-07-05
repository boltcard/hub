package admin

import (
	"card/phoenix"
	"card/web"
	"net/http"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

type paymentIn struct {
	PaymentInCards      []paymentInCard
	FirstPageEnabled    string
	PreviousPageEnabled string
	NextPageEnabled     string
	FirstPageLink       string
	PreviousPageLink    string
	NextPageLink        string
	CurrentPageNumber   string
}

type paymentInCard struct {
	CardStyle      string
	CardHeaderText string
	CardBodyText   string
}

// pagination format: https://domain_name/admin/payments-in/page/4/

func PaymentsIn(w http.ResponseWriter, r *http.Request) {

	const maxPaymentInCards = 24

	currentPage := 1
	var err error
	requestSplit := strings.Split(r.RequestURI, "/")
	if len(requestSplit) >= 5 {
		if requestSplit[3] == "page" {
			currentPage, err = strconv.Atoi(requestSplit[4])
			if err != nil {
				currentPage = 1
			}
		}
	}

	template_path := "/dist/pages/admin/payments-in/index.html"

	pmt_list, err := phoenix.ListIncomingPayments(maxPaymentInCards+1, maxPaymentInCards*(currentPage-1))
	if err != nil {
		log.Warn("phoenix error: ", err.Error())
	}

	template_data := paymentIn{
		FirstPageEnabled:    "disabled",
		PreviousPageEnabled: "disabled",
		NextPageEnabled:     "disabled",
		FirstPageLink:       "/admin/payments-in/page/1/",
		PreviousPageLink:    "/admin/payments-in/page/" + strconv.Itoa(currentPage-1) + "/",
		NextPageLink:        "/admin/payments-in/page/" + strconv.Itoa(currentPage+1) + "/",
		CurrentPageNumber:   strconv.Itoa(currentPage),
	}

	if currentPage > 1 {
		template_data.FirstPageEnabled = ""
		template_data.PreviousPageEnabled = ""
	}

	var numCards int

	if len(pmt_list) > maxPaymentInCards {
		numCards = maxPaymentInCards
		template_data.NextPageEnabled = ""
	} else {
		numCards = len(pmt_list)
	}

	for i := 0; i < numCards; i++ {

		c := paymentInCard{
			CardStyle:      "card-warning",
			CardHeaderText: time.Unix(0, pmt_list[i].CreatedAt*int64(time.Millisecond)).Format("Mon 2 Jan 2006 15:04 UTC"),
			CardBodyText:   "",
		}

		if pmt_list[i].IsPaid {
			c.CardStyle = "card-success"
		}

		//c.CardBodyText = pmt_list[i].Invoice

		template_data.PaymentInCards = append(template_data.PaymentInCards, c)
	}

	web.RenderHtmlFromTemplate(w, template_path, template_data)
}
