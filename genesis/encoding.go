package genesis

import (
	"encoding/json"
	"io/ioutil"

	"code.vegaprotocol.io/vega/assets"
	"code.vegaprotocol.io/vega/governance"
)

type GenesisState struct {
	Governance governance.GenesisState `json:"governance"`
	Assets     assets.GenesisState     `json:"assets"`
}

func DefaultGenesisState() GenesisState {
	return GenesisState{
		Governance: governance.DefaultGenesisState(),
		Assets:     assets.DefaultGenesisState(),
	}
}

func DumpDefault() (string, error) {
	gstate := DefaultGenesisState()
	bytes, err := json.MarshalIndent(&gstate, "  ", "  ")
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func UpdateInPlaceDefault(tmCfgPath string) error {
	tmCfgBytes, err := ioutil.ReadFile(tmCfgPath)
	if err != nil {
		return err
	}

	tmGenesis := map[string]interface{}{}
	err = json.Unmarshal(tmCfgBytes, &tmGenesis)
	if err != nil {
		return err
	}

	// make our raw message from the vega genesis state
	rawState, err := json.Marshal(DefaultGenesisState())
	if err != nil {
		return err
	}

	tmGenesis["app_state"] = json.RawMessage(rawState)
	tmCfgBytes, err = json.MarshalIndent(&tmGenesis, "  ", "  ")
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(tmCfgPath, tmCfgBytes, 0644)
	if err != nil {
		return err
	}
	return nil
}
