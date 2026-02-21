package phoenix

import (
	"encoding/json"
)

type NodeInfo struct {
	NodeID   string `json:"nodeId"`
	Version  string `json:"version"`
	Channels []struct {
		State               string `json:"state"`
		ChannelID           string `json:"channelId"`
		BalanceSat          int    `json:"balanceSat"`
		InboundLiquiditySat int    `json:"inboundLiquiditySat"`
		CapacitySat         int    `json:"capacitySat"`
		FundingTxID         string `json:"fundingTxId"`
	} `json:"channels"`
}

func GetNodeInfo() (NodeInfo, error) {
	var nodeInfo NodeInfo

	body, err := doGet("/getinfo", "GetNodeInfo")
	if err != nil {
		return nodeInfo, err
	}

	err = json.Unmarshal(body, &nodeInfo)
	if err != nil {
		return nodeInfo, err
	}

	return nodeInfo, nil
}
