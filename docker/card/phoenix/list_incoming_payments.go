package phoenix

import (
	"card/util"
	"strconv"

	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/go-ini/ini"
	log "github.com/sirupsen/logrus"
)

type IncomingPayments []struct {
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

func ListIncomingPayments(limit int, offset int) (IncomingPayments, error) {
	log.Info("listIncomingPayments")

	var incomingPayments IncomingPayments

	cfg, err := ini.Load("/root/.phoenix/phoenix.conf")
	util.Check(err)

	hp := cfg.Section("").Key("http-password").String()

	client := http.Client{Timeout: 5 * time.Second}

	req, err := http.NewRequest(http.MethodGet, "http://phoenix:9740/payments/incoming", http.NoBody)
	if err != nil {
		log.Fatal(err)
	}

	q := req.URL.Query()
	q.Add("limit", strconv.Itoa(limit))
	q.Add("offset", strconv.Itoa(offset))
	q.Add("all", "true") // include unpaid invoices
	req.URL.RawQuery = q.Encode()

	req.SetBasicAuth("", hp)

	res, err := client.Do(req)
	util.Check(err)

	defer res.Body.Close()

	resBody, err := io.ReadAll(res.Body)
	util.Check(err)

	if res.StatusCode != 200 {
		log.Warning("listIncomingPayments StatusCode ", res.StatusCode)
		return incomingPayments, errors.New("failed API call to Phoenix listIncomingPayments")
	}

	//log.Info(string(resBody))

	err = json.Unmarshal(resBody, &incomingPayments)
	util.Check(err)

	return incomingPayments, nil
}
