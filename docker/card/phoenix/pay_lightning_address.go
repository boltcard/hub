package phoenix

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

type PayLightningAddressRequest struct {
	AmountSat string
	Address   string
	Message   string
}

type PayLightningAddressResponse struct {
	RecipientAmountSat int    `json:"recipientAmountSat"`
	RoutingFeeSat      int    `json:"routingFeeSat"`
	PaymentId          string `json:"paymentId"`
	PaymentHash        string `json:"paymentHash"`
	PaymentPreimage    string `json:"paymentPreimage"`
	Reason             string `json:"reason"`
}

// PayLightningAddress pays a Lightning address (user@domain) via phoenixd's
// /paylnaddress endpoint. It mirrors SendLightningPayment: the second return
// value is a machine-readable reason string for the outcome.
func PayLightningAddress(
	payLightningAddressRequest PayLightningAddressRequest,
) (
	PayLightningAddressResponse,
	string,
	error,
) {
	var payLightningAddressResponse PayLightningAddressResponse

	password, err := getPassword()
	if err != nil {
		log.Warn(err)
		return payLightningAddressResponse,
			"no_config",
			errors.New("could not load config for PayLightningAddress")
	}

	formBody := url.Values{
		"amountSat": []string{payLightningAddressRequest.AmountSat},
		"address":   []string{payLightningAddressRequest.Address},
	}
	if payLightningAddressRequest.Message != "" {
		formBody.Set("message", payLightningAddressRequest.Message)
	}
	reader := strings.NewReader(formBody.Encode())

	req, err := http.NewRequest(http.MethodPost, phoenixBaseURL+"/paylnaddress", reader)
	if err != nil {
		log.Warn(err)
		return payLightningAddressResponse,
			"failed_request_creation",
			errors.New("could not create request for PayLightningAddress")
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth("", password)

	client := http.Client{Timeout: 30 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		// As with SendLightningPayment, a timeout is ambiguous — the payment
		// may or may not have been made and needs manual reconciliation.
		log.Error(err)
		return payLightningAddressResponse,
			"phoenix_api_timeout",
			errors.New("no response to PayLightningAddress")
	}

	defer res.Body.Close()

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		log.Error(err)
		return payLightningAddressResponse,
			"failed_read_response",
			errors.New("could not read response to PayLightningAddress")
	}

	if res.StatusCode != 200 {
		log.Warn("PayLightningAddress StatusCode ", res.StatusCode, " ResBody ", string(resBody))
		return payLightningAddressResponse,
			"fail_status_code",
			errors.New("fail status code returned for PayLightningAddress")
	}

	log.Info(string(resBody))

	err = json.Unmarshal(resBody, &payLightningAddressResponse)
	if err != nil {
		log.Error(err)
		return payLightningAddressResponse,
			"failed_decode_response",
			errors.New("could not decode response to PayLightningAddress")
	}

	return payLightningAddressResponse, "no_error", nil
}
