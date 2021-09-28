package genesis

import (
	"encoding/json"
	"io/ioutil"

	"code.vegaprotocol.io/vega/assets"
	"code.vegaprotocol.io/vega/blockchain/abci"
	"code.vegaprotocol.io/vega/checkpoint"
	vgjson "code.vegaprotocol.io/shared/libs/json"
	"code.vegaprotocol.io/vega/limits"
	"code.vegaprotocol.io/vega/netparams"
	"code.vegaprotocol.io/vega/validators"
)

type GenesisState struct {
	Assets     assets.GenesisState     `json:"assets"`
	Validators validators.GenesisState `json:"validators"`
	Network    abci.GenesisState       `json:"network"`
	NetParams  netparams.GenesisState  `json:"network_parameters"`
	Limits     limits.GenesisState     `json:"network_limits"`
	Checkpoint checkpoint.GenesisState `json:"checkpoint"`
}

func DefaultGenesisState() GenesisState {
	return GenesisState{
		Limits:     limits.DefaultGenesisState(),
		Assets:     assets.DefaultGenesisState(),
		Validators: validators.DefaultGenesisState(),
		Network:    abci.DefaultGenesis(),
		NetParams:  netparams.DefaultGenesisState(),
		Checkpoint: checkpoint.DefaultGenesisState(),
	}
}

func UpdateInPlace(gs *GenesisState, tmCfgPath string) error {
	tmCfgBytes, err := ioutil.ReadFile(tmCfgPath)
	if err != nil {
		return err
	}

	tmGenesis := map[string]interface{}{}
	if err := json.Unmarshal(tmCfgBytes, &tmGenesis); err != nil {
		return err
	}

	// make our raw message from the vega genesis state
	rawState, err := json.Marshal(gs)
	if err != nil {
		return err
	}

	tmGenesis["app_state"] = json.RawMessage(rawState)
	tmCfgBytes, err = vgjson.Prettify(&tmGenesis)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(tmCfgPath, tmCfgBytes, 0644)
}
