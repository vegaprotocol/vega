package validators

import "encoding/json"

type GenesisState ValidatorMapping

func DefaultGenesisState() GenesisState {
	return GenesisState(ValidatorMapping{})
}

func LoadGenesisState(bytes []byte) (GenesisState, error) {
	state := struct {
		Validators GenesisState `json:"validators"`
	}{}

	return state.Validators, json.Unmarshal(bytes, &state)
}
