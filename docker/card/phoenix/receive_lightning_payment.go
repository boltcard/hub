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

type ReceiveLightningPaymentRequest struct {
	Description string
	AmountSat   string
	ExternalId  string
}

type ReceiveLightningPaymentResponse struct {
	AmountSat   int    `json:"amountSat"`
	PaymentHash string `json:"paymentHash"`
	Serialized  string `json:"serialized"`
}

func ReceiveLightningPayment(receiveLightningPaymentRequest ReceiveLightningPaymentRequest) (ReceiveLightningPaymentResponse, error) {

	var receiveLightningPaymentResponse ReceiveLightningPaymentResponse

	cfg, err := ini.Load("/root/.phoenix/phoenix.conf")
	util.Check(err)

	hp := cfg.Section("").Key("http-password").String()

	client := http.Client{Timeout: 5 * time.Second}

	formBody := url.Values{
		"description": []string{receiveLightningPaymentRequest.Description},
		"amountSat":   []string{receiveLightningPaymentRequest.AmountSat},
		"externalId":  []string{receiveLightningPaymentRequest.ExternalId},
	}
	dataReader := formBody.Encode()
	reader := strings.NewReader(dataReader)

	req, err := http.NewRequest(http.MethodPost, "http://phoenix:9740/createinvoice", reader)
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
		log.Warning("ReceiveLightningPayment StatusCode ", res.StatusCode)
		return receiveLightningPaymentResponse, errors.New("failed API call to Phoenix ReceiveLightningPayment")
	}

	log.Info(string(resBody))

	err = json.Unmarshal(resBody, &receiveLightningPaymentResponse)
	util.Check(err)

	return receiveLightningPaymentResponse, nil
}
