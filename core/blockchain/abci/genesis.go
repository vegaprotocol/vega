// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package abci

import (
	"encoding/json"
	"errors"
)

var ErrNoNetworkGenesisState = errors.New("no network genesis state")

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
