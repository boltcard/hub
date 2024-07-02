package phoenix

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/go-ini/ini"
	log "github.com/sirupsen/logrus"
)

type Balance struct {
	BalanceSat   int `json:"balanceSat"`
	FeeCreditSat int `json:"feeCreditSat"`
}

func GetBalance() (Balance, error) {
	var balance Balance

	cfg, err := ini.Load("/root/.phoenix/phoenix.conf")
	if err != nil {
		return balance, err
	}

	hp := cfg.Section("").Key("http-password").String()

	client := http.Client{Timeout: 5 * time.Second}

	req, err := http.NewRequest(http.MethodGet, "http://phoenix:9740/getbalance", http.NoBody)
	if err != nil {
		return balance, err
	}

	req.SetBasicAuth("", hp)

	res, err := client.Do(req)
	if err != nil {
		return balance, err
	}

	defer res.Body.Close()

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return balance, err
	}

	if res.StatusCode != 200 {
		log.Warning("GetBalance StatusCode ", res.StatusCode)
		return balance, errors.New("failed API call to Phoenix GetBalance")
	}

	//log.Info(string(resBody))

	err = json.Unmarshal(resBody, &balance)
	if err != nil {
		return balance, err
	}

	return balance, nil
}
