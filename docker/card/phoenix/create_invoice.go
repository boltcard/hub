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
	log.Info("createInvoice")

	var createInvoiceResponse CreateInvoiceResponse

	cfg, err := ini.Load("/root/.phoenix/phoenix.conf")
	util.Check(err)

	hp := cfg.Section("").Key("http-password").String()

	client := http.Client{Timeout: 5 * time.Second}

	formBody := url.Values{
		"description": []string{createInvoiceRequest.Description},
		"amountSat":   []string{createInvoiceRequest.AmountSat},
		"externalId":  []string{createInvoiceRequest.ExternalId},
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
		log.Warning("createinvoice StatusCode ", res.StatusCode)
		return createInvoiceResponse, errors.New("failed API call to Phoenix createinvoice")
	}

	log.Info(string(resBody))

	err = json.Unmarshal(resBody, &createInvoiceResponse)
	util.Check(err)

	return createInvoiceResponse, nil
}
