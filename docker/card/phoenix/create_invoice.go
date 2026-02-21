package phoenix

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	log "github.com/sirupsen/logrus"
)

type CreateInvoiceRequest struct {
	Description string
	AmountSat   string
	ExternalId  string
}

type CreateInvoiceResponse struct {
	AmountSat   int    `json:"amountSat"`
	PaymentHash string `json:"paymentHash"`
	Serialized  string `json:"serialized"`
}

func CreateInvoice(createInvoiceRequest CreateInvoiceRequest) (CreateInvoiceResponse, error) {
	var createInvoiceResponse CreateInvoiceResponse

	formBody := url.Values{
		"description": []string{createInvoiceRequest.Description},
		"amountSat":   []string{createInvoiceRequest.AmountSat},
		"externalId":  []string{createInvoiceRequest.ExternalId},
	}
	reader := strings.NewReader(formBody.Encode())

	req, err := http.NewRequest(http.MethodPost, phoenixBaseURL+"/createinvoice", reader)
	if err != nil {
		return createInvoiceResponse, err
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	body, err := doRequest(req, defaultTimeout, "CreateInvoice")
	if err != nil {
		return createInvoiceResponse, err
	}

	log.Info(string(body))

	err = json.Unmarshal(body, &createInvoiceResponse)
	if err != nil {
		return createInvoiceResponse, err
	}

	return createInvoiceResponse, nil
}
