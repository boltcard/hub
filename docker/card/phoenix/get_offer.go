package phoenix

import (
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/go-ini/ini"
	log "github.com/sirupsen/logrus"
)

func GetOffer() (offer string, err error) {

	cfg, err := ini.Load("/root/.phoenix/phoenix.conf")
	if err != nil {
		return "", err
	}

	hp := cfg.Section("").Key("http-password").String()

	client := http.Client{Timeout: 5 * time.Second}

	req, err := http.NewRequest(http.MethodGet, "http://phoenix:9740/getoffer", http.NoBody)
	if err != nil {
		return "", err
	}

	req.SetBasicAuth("", hp)

	res, err := client.Do(req)
	if err != nil {
		return "", err
	}

	defer res.Body.Close()

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	if res.StatusCode != 200 {
		log.Warning("GetBalance StatusCode ", res.StatusCode)
		return "", errors.New("failed API call to Phoenix GetOffer")
	}

	offer = string(resBody)

	return offer, nil
}
