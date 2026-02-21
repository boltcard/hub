package phoenix

import (
	"encoding/json"
	"fmt"
	"net/url"
)

type OutgoingPaymentResponse struct {
	PaymentHash string `json:"paymentHash"`
	Preimage    string `json:"preimage"`
	IsPaid      bool   `json:"isPaid"`
	SentSat     int    `json:"sent"`
	FeesSat     int    `json:"fees"`
	Invoice     string `json:"invoice"`
	CompletedAt int64  `json:"completedAt"`
	CreatedAt   int64  `json:"createdAt"`
}

func GetOutgoingPayment(PaymentId string) (OutgoingPaymentResponse, error) {
	var outgoingPaymentResponse OutgoingPaymentResponse

	path := fmt.Sprintf("/payments/outgoing/%s", url.QueryEscape(PaymentId))
	body, err := doGet(path, "GetOutgoingPayment")
	if err != nil {
		return outgoingPaymentResponse, err
	}

	err = json.Unmarshal(body, &outgoingPaymentResponse)
	if err != nil {
		return outgoingPaymentResponse, err
	}

	return outgoingPaymentResponse, nil
}
