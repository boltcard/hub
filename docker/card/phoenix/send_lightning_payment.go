package phoenix

import (
	"card/util"

	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-ini/ini"
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
}

func SendLightningPayment(sendLightningPaymentRequest SendLightningPaymentRequest) (SendLightningPaymentResponse, error) {

	var sendLightningPaymentResponse SendLightningPaymentResponse

	cfg, err := ini.Load("/root/.phoenix/phoenix.conf")
	util.Check(err)

	hp := cfg.Section("").Key("http-password").String()

	client := http.Client{Timeout: 5 * time.Second}

	formBody := url.Values{
		"amountSat": []string{sendLightningPaymentRequest.AmountSat},
		"invoice":   []string{sendLightningPaymentRequest.Invoice},
	}
	dataReader := formBody.Encode()
	reader := strings.NewReader(dataReader)

	req, err := http.NewRequest(http.MethodPost, "http://phoenix:9740/payinvoice", reader)
	if err != nil {
		log.Fatal(err)
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	req.SetBasicAuth("", hp)

	res, err := client.Do(req)
	util.Check(err)

	defer res.Body.Close()

	resBody, err := io.ReadAll(res.Body)
	util.Check(err)

	if res.StatusCode != 200 {
		log.Warning("SendLightningPayment StatusCode ", res.StatusCode)
		return sendLightningPaymentResponse, errors.New("failed API call to Phoenix SendLightningPayment")
	}

	log.Info(string(resBody))

	err = json.Unmarshal(resBody, &sendLightningPaymentResponse)
	util.Check(err)

	return sendLightningPaymentResponse, nil
}
