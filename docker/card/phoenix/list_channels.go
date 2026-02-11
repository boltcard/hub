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

type Channel struct {
	State             string
	ChannelID         string
	BalanceMsat       int64
	InboundLiquidMsat int64
}

type listChannelRaw struct {
	Type        string          `json:"type"`
	State       json.RawMessage `json:"state,omitempty"`
	Commitments json.RawMessage `json:"commitments,omitempty"`
}

type commitmentsData struct {
	ChannelParams struct {
		ChannelID string `json:"channelId"`
	} `json:"channelParams"`
	Active []struct {
		LocalCommit struct {
			Spec struct {
				ToLocal  int64 `json:"toLocal"`
				ToRemote int64 `json:"toRemote"`
			} `json:"spec"`
		} `json:"localCommit"`
	} `json:"active"`
}

func extractChannel(raw listChannelRaw) (Channel, bool) {
	var ch Channel

	// determine state name from the type field
	typeName := raw.Type
	// extract simple name from qualified name e.g. "fr.acinq.lightning.channel.states.Normal" -> "Normal"
	for i := len(typeName) - 1; i >= 0; i-- {
		if typeName[i] == '.' {
			typeName = typeName[i+1:]
			break
		}
	}

	// for Offline/Syncing, the commitments are nested inside "state"
	commitmentsJSON := raw.Commitments
	if (typeName == "Offline" || typeName == "Syncing") && raw.State != nil {
		var nested listChannelRaw
		if err := json.Unmarshal(raw.State, &nested); err == nil {
			commitmentsJSON = nested.Commitments
			// use the inner state's type for display
			innerType := nested.Type
			for i := len(innerType) - 1; i >= 0; i-- {
				if innerType[i] == '.' {
					innerType = innerType[i+1:]
					break
				}
			}
			typeName = innerType
		}
	}

	ch.State = typeName

	if commitmentsJSON == nil {
		return ch, false
	}

	var commitments commitmentsData
	if err := json.Unmarshal(commitmentsJSON, &commitments); err != nil {
		return ch, false
	}

	ch.ChannelID = commitments.ChannelParams.ChannelID

	if len(commitments.Active) > 0 {
		spec := commitments.Active[0].LocalCommit.Spec
		ch.BalanceMsat = spec.ToLocal
		ch.InboundLiquidMsat = spec.ToRemote
	}

	return ch, true
}

func ListChannels() ([]Channel, error) {

	cfg, err := ini.Load("/root/.phoenix/phoenix.conf")
	if err != nil {
		return nil, err
	}

	hp := cfg.Section("").Key("http-password").String()

	client := http.Client{Timeout: 5 * time.Second}

	req, err := http.NewRequest(http.MethodGet, "http://phoenix:9740/listchannels", http.NoBody)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth("", hp)

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != 200 {
		log.Warning("ListChannels StatusCode ", res.StatusCode)
		return nil, errors.New("failed API call to Phoenix ListChannels")
	}

	var rawChannels []listChannelRaw
	err = json.Unmarshal(resBody, &rawChannels)
	if err != nil {
		return nil, err
	}

	var channels []Channel
	for _, raw := range rawChannels {
		if ch, ok := extractChannel(raw); ok {
			channels = append(channels, ch)
		}
	}

	return channels, nil
}
