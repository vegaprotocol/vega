package genesis

import (
	"encoding/json"

	tmtypes "github.com/tendermint/tendermint/types"
)

func GenesisFromJSON(rawGenesisDoc []byte) (*tmtypes.GenesisDoc, *GenesisState, error) {
	genesisDoc, err := tmtypes.GenesisDocFromJSON(rawGenesisDoc)
	if err != nil {
		return nil, nil, err
	}

	genesisState := &GenesisState{}
	err = json.Unmarshal(genesisDoc.AppState, genesisState)
	if err != nil {
		return nil, nil, err
	}
	return genesisDoc, genesisState, nil
}
