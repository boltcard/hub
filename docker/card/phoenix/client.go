package phoenix

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/go-ini/ini"
	log "github.com/sirupsen/logrus"
)

const phoenixBaseURL = "http://phoenix:9740"
const defaultTimeout = 5 * time.Second

// loadPassword reads the http-password from the phoenix config file.
func loadPassword() (string, error) {
	cfg, err := ini.Load("/root/.phoenix/phoenix.conf")
	if err != nil {
		return "", fmt.Errorf("load phoenix config: %w", err)
	}
	return cfg.Section("").Key("http-password").String(), nil
}

// doRequest executes an HTTP request against the Phoenix API with basic auth.
// It returns the response body bytes on success, or an error on failure.
func doRequest(req *http.Request, timeout time.Duration, endpointName string) ([]byte, error) {
	password, err := loadPassword()
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth("", password)

	client := http.Client{Timeout: timeout}
	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s request failed: %w", endpointName, err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("%s read body: %w", endpointName, err)
	}

	if res.StatusCode != 200 {
		log.Warn(endpointName, " StatusCode ", res.StatusCode)
		return nil, errors.New("failed API call to Phoenix " + endpointName)
	}

	return body, nil
}

// doGet is a convenience wrapper for GET requests with the default timeout.
func doGet(path string, endpointName string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, phoenixBaseURL+path, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("%s create request: %w", endpointName, err)
	}
	return doRequest(req, defaultTimeout, endpointName)
}
