package netparams

import (
	"encoding/json"
	"errors"
)

var (
	ErrNoNetParamsGenesisState = errors.New("no network parameters genesis state")
)

type GenesisState map[string]string

func DefaultGenesisState() GenesisState {
	state := map[string]string{}
	netp := defaultNetParams()
	for k, v := range netp {
		state[k] = v.String()
	}

	return state
}

func LoadGenesisState(bytes []byte) (GenesisState, error) {
	state := struct {
		NetParams *GenesisState `json:"network_parameters"`
	}{}
	err := json.Unmarshal(bytes, &state)
	if err != nil {
		return nil, err
	}
	if state.NetParams == nil {
		return nil, ErrNoNetParamsGenesisState
	}
	return *state.NetParams, nil
}
