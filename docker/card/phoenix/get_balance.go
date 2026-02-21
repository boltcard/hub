package phoenix

import (
	"encoding/json"
)

type Balance struct {
	BalanceSat   int `json:"balanceSat"`
	FeeCreditSat int `json:"feeCreditSat"`
}

func GetBalance() (Balance, error) {
	var balance Balance

	body, err := doGet("/getbalance", "GetBalance")
	if err != nil {
		return balance, err
	}

	err = json.Unmarshal(body, &balance)
	if err != nil {
		return balance, err
	}

	return balance, nil
}
