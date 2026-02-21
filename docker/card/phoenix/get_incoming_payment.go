package phoenix

import (
	"encoding/json"
	"fmt"
	"net/url"
)

type IncomingPayment struct {
	PaymentHash string `json:"paymentHash"`
	Preimage    string `json:"preimage"`
	ExternalID  string `json:"externalId"`
	Description string `json:"description"`
	Invoice     string `json:"invoice"`
	IsPaid      bool   `json:"isPaid"`
	ReceivedSat int    `json:"receivedSat"`
	Fees        int    `json:"fees"`
	CompletedAt int64  `json:"completedAt"`
	CreatedAt   int64  `json:"createdAt"`
}

func GetIncomingPayment(PaymentHash string) (IncomingPayment, error) {
	var incomingPayment IncomingPayment

	path := fmt.Sprintf("/payments/incoming/%s", url.QueryEscape(PaymentHash))
	body, err := doGet(path, "GetIncomingPayment")
	if err != nil {
		return incomingPayment, err
	}

	err = json.Unmarshal(body, &incomingPayment)
	if err != nil {
		return incomingPayment, err
	}

	return incomingPayment, nil
}
