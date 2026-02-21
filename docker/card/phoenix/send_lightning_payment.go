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

type SendLightningPaymentRequest struct {
	AmountSat string
	Invoice   string
}

type SendLightningPaymentResponse struct {
	RecipientAmountSat int    `json:"recipientAmountSat"`
	RoutingFeeSat      int    `json:"routingFeeSat"`
	PaymentId          string `json:"paymentId"`
	PaymentHash        string `json:"paymentHash"`
	PaymentPreimage    string `json:"paymentPreimage"`
	Reason             string `json:"reason"`
}

// TODO:
// log timeout (should never happen but we need a timeout)
// log success
// log fail with reason in database
// add these records to the admin dashboard
func SendLightningPayment(
	sendLightningPaymentRequest SendLightningPaymentRequest,
) (
	SendLightningPaymentResponse,
	string,
	error,
) {
	var sendLightningPaymentResponse SendLightningPaymentResponse

	password, err := loadPassword()
	if err != nil {
		log.Warning(err)
		return sendLightningPaymentResponse,
			"no_config",
			errors.New("could not load config for SendLightningPayment")
	}

	formBody := url.Values{
		"amountSat": []string{sendLightningPaymentRequest.AmountSat},
		"invoice":   []string{sendLightningPaymentRequest.Invoice},
	}
	reader := strings.NewReader(formBody.Encode())

	req, err := http.NewRequest(http.MethodPost, phoenixBaseURL+"/payinvoice", reader)
	if err != nil {
		log.Warning(err)
		return sendLightningPaymentResponse,
			"failed_request_creation",
			errors.New("could not create request for SendLightningPayment")
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth("", password)

	client := http.Client{Timeout: 30 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		// we never want this error as the payment may or may not be made
		// so we assume it has been made - it needs to be manually corrected
		// and the cause of the error established (probably timeout too short)
		// i.e.
		//  ERRO[0000] Post "http://phoenix:9740/payinvoice": context deadline exceeded (Client.Timeout exceeded while awaiting headers)
		log.Error(err)
		return sendLightningPaymentResponse,
			"phoenix_api_timeout",
			errors.New("no response to SendLightningPayment")
	}

	defer res.Body.Close()

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		log.Error(err)
		return sendLightningPaymentResponse,
			"failed_read_response",
			errors.New("could not read response to SendLightningPayment")
	}

	if res.StatusCode != 200 {
		log.Warning("SendLightningPayment StatusCode ", res.StatusCode, "ResBody", string(resBody))
		return sendLightningPaymentResponse,
			"fail_status_code",
			errors.New("fail status code returned for SendLightningPayment")
	}

	log.Info(string(resBody))

	err = json.Unmarshal(resBody, &sendLightningPaymentResponse)
	if err != nil {
		log.Error(err)
		return sendLightningPaymentResponse,
			"failed_decode_response",
			errors.New("could not decode response to SendLightningPayment")
	}

	return sendLightningPaymentResponse, "no_error", nil
}
