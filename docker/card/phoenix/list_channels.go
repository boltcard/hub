package phoenix

import (
	"encoding/json"
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
	body, err := doGet("/listchannels", "ListChannels")
	if err != nil {
		return nil, err
	}

	var rawChannels []listChannelRaw
	err = json.Unmarshal(body, &rawChannels)
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
