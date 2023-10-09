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

package limits

import (
	"encoding/json"
	"errors"
)

var ErrNoLimitsGenesisState = errors.New("no limits genesis state")

type GenesisState struct {
	ProposeMarketEnabled bool `json:"propose_market_enabled"`
	ProposeAssetEnabled  bool `json:"propose_asset_enabled"`
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
