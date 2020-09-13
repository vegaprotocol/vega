package genesis

import (
	"encoding/json"
	"io/ioutil"

	"code.vegaprotocol.io/vega/assets"
	"code.vegaprotocol.io/vega/governance"
	"code.vegaprotocol.io/vega/nodewallet"
	"code.vegaprotocol.io/vega/validators"
)

type GenesisState struct {
	Governance governance.GenesisState `json:"governance"`
	Assets     assets.GenesisState     `json:"assets"`
	Validators validators.GenesisState `json:"validators"`
	NodeWallet nodewallet.GenesisState `json:"node_wallet"`
}

func DefaultGenesisState() GenesisState {
	return GenesisState{
		Governance: governance.DefaultGenesisState(),
		Assets:     assets.DefaultGenesisState(),
		Validators: validators.DefaultGenesisState(),
		NodeWallet: nodewallet.DefaultGenesisState(),
	}
}

func DumpDefault() (string, error) {
	gstate := DefaultGenesisState()
	return Dump(&gstate)
}

func Dump(s *GenesisState) (string, error) {
	bytes, err := json.MarshalIndent(s, "  ", "  ")
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func UpdateInPlaceDefault(tmCfgPath string) error {
	gs := DefaultGenesisState()
	return UpdateInPlace(&gs, tmCfgPath)
}

func UpdateInPlace(gs *GenesisState, tmCfgPath string) error {
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
	rawState, err := json.Marshal(gs)
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
