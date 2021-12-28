package netparams

import (
	"encoding/json"
	"errors"
)

var ErrNoNetParamsGenesisState = errors.New("no network parameters genesis state")

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

type GenesisStateOverwrite []string

func DefaultGenesisStateOverwrite() GenesisStateOverwrite {
	state := []string{}
	return state
}

func LoadGenesisStateOverwrite(bytes []byte) (GenesisStateOverwrite, error) {
	state := struct {
		NetParamsOverwrite *GenesisStateOverwrite `json:"network_parameters_checkpoint_overwrite"`
	}{}
	err := json.Unmarshal(bytes, &state)
	if err != nil {
		return nil, err
	}
	if state.NetParamsOverwrite == nil {
		return nil, nil // not an error, not mandatory to have overwrite list
	}
	return *state.NetParamsOverwrite, nil
}
