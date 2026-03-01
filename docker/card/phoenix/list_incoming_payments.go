package phoenix

import (
	"encoding/json"
	"net/http"
	"strconv"
)

type IncomingPayments []struct {
	PaymentHash string `json:"paymentHash"`
	Preimage    string `json:"preimage"`
	ExternalID  string `json:"externalId"`
	Description string `json:"description"`
	Invoice     string `json:"invoice"`
	IsPaid      bool   `json:"isPaid"`
	ReceivedSat int    `json:"receivedSat"`
	Fees        int    `json:"fees"`
	PayerNote   string `json:"payerNote"`
	CompletedAt int64  `json:"completedAt"`
	CreatedAt   int64  `json:"createdAt"`
}

func ListIncomingPayments(limit int, offset int) (IncomingPayments, error) {
	var incomingPayments IncomingPayments

	req, err := http.NewRequest(http.MethodGet, phoenixBaseURL+"/payments/incoming", http.NoBody)
	if err != nil {
		return incomingPayments, err
	}

	q := req.URL.Query()
	q.Add("limit", strconv.Itoa(limit))
	q.Add("offset", strconv.Itoa(offset))
	q.Add("all", "true") // include unpaid invoices
	req.URL.RawQuery = q.Encode()

	body, err := doRequest(req, defaultTimeout, "ListIncomingPayments")
	if err != nil {
		return incomingPayments, err
	}

	err = json.Unmarshal(body, &incomingPayments)
	if err != nil {
		return incomingPayments, err
	}

	return incomingPayments, nil
}
