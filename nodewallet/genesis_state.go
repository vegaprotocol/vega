package nodewallet

import "encoding/json"

type GenesisState struct {
	ETH ETHGenesisState `json:"eth_genesis_state"`
}

type ETHGenesisState struct {
	ChainID            string `json:"chain_id"`
	ERC20BridgeAddress string `json:"erc20_bridge_address"`
}

func DefaultGenesisState() GenesisState {
	return GenesisState{
		// generate ropsten settings
		ETH: ETHGenesisState{
			ChainID:            "",
			ERC20BridgeAddress: "",
		},
	}
}

func LoadGenesisState(bytes []byte) (*GenesisState, error) {
	state := struct {
		NodeWallet *GenesisState `json:"node_wallet"`
	}{}
	err := json.Unmarshal(bytes, &state)
	if err != nil {
		return nil, err
	}
	return state.NodeWallet, nil
}
