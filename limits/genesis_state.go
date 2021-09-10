package limits

import (
	"encoding/json"
	"errors"
	"time"
)

var (
	ErrNoLimitsGenesisState = errors.New("no limits genesis state")
)

type GenesisState struct {
	ProposeMarketEnabled     bool       `json:"propose_market_enabled"`
	ProposeAssetEnabled      bool       `json:"propose_asset_enabled"`
	ProposeMarketEnabledFrom *time.Time `json:"propose_market_enabled_from,omitempty"`
	ProposeAssetEnabledFrom  *time.Time `json:"propose_asset_enabled_from,omitempty"`
	BootstrapBlockCount      uint16     `json:"bootstrap_block_count"`
}

func DefaultGenesisState() GenesisState {
	return GenesisState{
		ProposeMarketEnabled: true,
		ProposeAssetEnabled:  true,
	}
}

func LoadGenesisState(bytes []byte) (*GenesisState, error) {
	state := struct {
		Limits *GenesisState `json:"network_limits"`
	}{}
	err := json.Unmarshal(bytes, &state)
	if err != nil {
		return nil, err
	}
	if state.Limits == nil {
		return nil, ErrNoLimitsGenesisState
	}

	return state.Limits, nil
}
