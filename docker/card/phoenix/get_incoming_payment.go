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

	cfg, err := ini.Load("/root/.phoenix/phoenix.conf")
	util.CheckAndPanic(err)

	hp := cfg.Section("").Key("http-password").String()

	client := http.Client{Timeout: 5 * time.Second}

	url := fmt.Sprintf("http://phoenix:9740/payments/incoming/%s", url.QueryEscape(PaymentHash))
	req, err := http.NewRequest(http.MethodGet, url, http.NoBody)
	if err != nil {
		log.Fatal(err)
	}

	req.SetBasicAuth("", hp)

	res, err := client.Do(req)
	util.CheckAndPanic(err)

	defer res.Body.Close()

	resBody, err := io.ReadAll(res.Body)
	util.CheckAndPanic(err)

	if res.StatusCode != 200 {
		log.Warning("GetIncomingPayment StatusCode ", res.StatusCode)
		return incomingPayment, errors.New("failed API call to Phoenix GetIncomingPayment")
	}

	//log.Info(string(resBody))

	err = json.Unmarshal(resBody, &incomingPayment)
	util.CheckAndPanic(err)

	return incomingPayment, nil
}
