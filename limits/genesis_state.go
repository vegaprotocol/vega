// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package limits

import (
	"encoding/json"
	"errors"
)

var ErrNoLimitsGenesisState = errors.New("no limits genesis state")

type GenesisState struct {
	ProposeMarketEnabled bool   `json:"propose_market_enabled"`
	ProposeAssetEnabled  bool   `json:"propose_asset_enabled"`
	BootstrapBlockCount  uint16 `json:"bootstrap_block_count"`
}

func DefaultGenesisState() GenesisState {
	return GenesisState{
		ProposeMarketEnabled: true,
		ProposeAssetEnabled:  true,
	}
}

func LoadGenesisState(bytes []byte) (*GenesisState, error) {
	state := struct {
		Limits *GenesisState `json:"network_limits"`
	}{}
	err := json.Unmarshal(bytes, &state)
	if err != nil {
		return nil, err
	}
	if state.Limits == nil {
		return nil, ErrNoLimitsGenesisState
	}

	return state.Limits, nil
}
