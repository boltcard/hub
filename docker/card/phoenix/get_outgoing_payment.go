package phoenix

import (
	"card/util"

	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/go-ini/ini"
	log "github.com/sirupsen/logrus"
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
	//log.Info("getOutgoingPayment")

	var outgoingPaymentResponse OutgoingPaymentResponse

	cfg, err := ini.Load("/root/.phoenix/phoenix.conf")
	util.Check(err)

	hp := cfg.Section("").Key("http-password").String()

	client := http.Client{Timeout: 5 * time.Second}

	url := fmt.Sprintf("http://phoenix:9740/payments/outgoing/%s", url.QueryEscape(PaymentId))
	req, err := http.NewRequest(http.MethodGet, url, http.NoBody)
	if err != nil {
		log.Fatal(err)
	}

	req.SetBasicAuth("", hp)

	res, err := client.Do(req)
	util.Check(err)

	defer res.Body.Close()

	resBody, err := io.ReadAll(res.Body)
	util.Check(err)

	if res.StatusCode != 200 {
		log.Warning("getOutgoingPayment StatusCode ", res.StatusCode)
		return outgoingPaymentResponse, errors.New("failed API call to Phoenix getOutgoingPayment")
	}

	//log.Info(string(resBody))

	err = json.Unmarshal(resBody, &outgoingPaymentResponse)
	util.Check(err)

	return outgoingPaymentResponse, nil
}
