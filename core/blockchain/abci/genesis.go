// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

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
