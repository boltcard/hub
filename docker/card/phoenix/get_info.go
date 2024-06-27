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

type Info struct {
	NodeID   string `json:"nodeId"`
	Channels []struct {
		State               string `json:"state"`
		ChannelID           string `json:"channelId"`
		BalanceSat          int    `json:"balanceSat"`
		InboundLiquiditySat int    `json:"inboundLiquiditySat"`
		CapacitySat         int    `json:"capacitySat"`
		FundingTxID         string `json:"fundingTxId"`
	} `json:"channels"`
}

func GetInfo() (Info, error) {
	var info Info

	cfg, err := ini.Load("/root/.phoenix/phoenix.conf")
	if err != nil {
		return info, err
	}

	hp := cfg.Section("").Key("http-password").String()

	client := http.Client{Timeout: 5 * time.Second}

	req, err := http.NewRequest(http.MethodGet, "http://phoenix:9740/getinfo", http.NoBody)
	if err != nil {
		return info, err
	}

	req.SetBasicAuth("", hp)

	res, err := client.Do(req)
	if err != nil {
		return info, err
	}

	defer res.Body.Close()

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return info, err
	}

	if res.StatusCode != 200 {
		log.Warning("getbalance StatusCode ", res.StatusCode)
		return info, errors.New("failed API call to Phoenix getbalance")
	}

	//log.Info(string(resBody))

	err = json.Unmarshal(resBody, &info)
	if err != nil {
		return info, err
	}

	return info, nil
}
