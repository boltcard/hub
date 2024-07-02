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

type OutgoingPayments []struct {
	PaymentID   string `json:"paymentId"`
	PaymentHash string `json:"paymentHash"`
	Preimage    string `json:"preimage"`
	IsPaid      bool   `json:"isPaid"`
	Sent        int    `json:"sent"`
	Fees        int    `json:"fees"`
	Invoice     string `json:"invoice"`
	CompletedAt int64  `json:"completedAt"`
	CreatedAt   int64  `json:"createdAt"`
}

func ListOutgoingPayments(limit int, offset int) (OutgoingPayments, error) {

	var outgoingPayments OutgoingPayments

	cfg, err := ini.Load("/root/.phoenix/phoenix.conf")
	util.Check(err)

	hp := cfg.Section("").Key("http-password").String()

	client := http.Client{Timeout: 5 * time.Second}

	req, err := http.NewRequest(http.MethodGet, "http://phoenix:9740/payments/outgoing", http.NoBody)
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
		log.Warning("ListOutgoingPayments StatusCode ", res.StatusCode)
		return outgoingPayments, errors.New("failed API call to Phoenix ListOutgoingPayments")
	}

	//log.Info(string(resBody))

	err = json.Unmarshal(resBody, &outgoingPayments)
	util.Check(err)

	return outgoingPayments, nil
}
