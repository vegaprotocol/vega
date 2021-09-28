package checkpoint

import "encoding/json"

type GenesisState struct {
	CheckpointHash string `json:"load_hash"`
}

func DefaultGenesisState() GenesisState {
	return GenesisState{} // default no hash
}

func LoadGenesisState(data []byte) (*GenesisState, error) {
	cp := &struct {
		Checkpoint *GenesisState `json:"checkpoint"`
	}{}
	if err := json.Unmarshal(data, &cp); err != nil {
		return nil, err
	}
	return cp.Checkpoint, nil
}
