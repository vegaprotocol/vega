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
			ChainID:            "3",
			ERC20BridgeAddress: "0xf6C9d3e937fb2dA4995272C1aC3f3D466B7c23fC",
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
