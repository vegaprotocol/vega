package abci

import (
	"encoding/json"
	"errors"
)

var (
	ErrNoNetworkGenesisState = errors.New("no network genesis state")
)

type GenesisState struct {
	// ReplayAttackThreshold protects the network against replay attacks. It sets a
	// toleration thershold between the current block in the chain and the block
	// heigh specified in the Tx.  Tx with blocks height >= than (chain's height -
	// distance) are rejected with a AbciTxnRejected.  It also keeps a ring-buffer
	// to cache seen Tx. The Ring buffer size defines the number of block to cache,
	// each block can hold an unlimited number of Txs.
	ReplayAttackThreshold uint `json:"replay_attack_threshold"`
}

func DefaultGenesis() GenesisState {
	return GenesisState{
		ReplayAttackThreshold: 150,
	}
}

func LoadGenesisState(bytes []byte) (*GenesisState, error) {
	state := struct {
		Network *GenesisState `json:"network"`
	}{}
	if err := json.Unmarshal(bytes, &state); err != nil {
		return nil, err
	}

	if state.Network == nil {
		return nil, ErrNoNetworkGenesisState
	}

	return state.Network, nil
}
