package phoenix

import (
	"encoding/json"
	"net/http"
	"strconv"
)

type OutgoingPayments []struct {
	PaymentID   string `json:"paymentId"`
	PaymentHash string `json:"paymentHash"`
	Preimage    string `json:"preimage"`
	IsPaid      bool   `json:"isPaid"`
	Sent        int    `json:"sent"`
	Fees        int    `json:"fees"`
	Invoice     string `json:"invoice"`
	CompletedAt int64  `json:"completedAt"`
	CreatedAt   int64  `json:"createdAt"`
}

func ListOutgoingPayments(limit int, offset int) (OutgoingPayments, error) {
	var outgoingPayments OutgoingPayments

	req, err := http.NewRequest(http.MethodGet, phoenixBaseURL+"/payments/outgoing", http.NoBody)
	if err != nil {
		return outgoingPayments, err
	}

	q := req.URL.Query()
	q.Add("limit", strconv.Itoa(limit))
	q.Add("offset", strconv.Itoa(offset))
	q.Add("all", "true") // include unpaid invoices
	req.URL.RawQuery = q.Encode()

	body, err := doRequest(req, defaultTimeout, "ListOutgoingPayments")
	if err != nil {
		return outgoingPayments, err
	}

	err = json.Unmarshal(body, &outgoingPayments)
	if err != nil {
		return outgoingPayments, err
	}

	return outgoingPayments, nil
}
