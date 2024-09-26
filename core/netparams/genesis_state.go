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

package netparams

import (
	"encoding/json"
	"errors"
)

var ErrNoNetParamsGenesisState = errors.New("no network parameters genesis state")

type GenesisState map[string]string

func DefaultGenesisState() GenesisState {
	state := map[string]string{}
	netp := defaultNetParams()

	for k, v := range netp {
		state[k] = v.String()
	}

	return state
}

func LoadGenesisState(bytes []byte) (GenesisState, error) {
	state := struct {
		NetParams *GenesisState `json:"network_parameters"`
	}{}
	err := json.Unmarshal(bytes, &state)
	if err != nil {
		return nil, err
	}
	if state.NetParams == nil {
		return nil, ErrNoNetParamsGenesisState
	}
	return *state.NetParams, nil
}

type GenesisStateOverwrite []string

func LoadGenesisStateOverwrite(bytes []byte) (GenesisStateOverwrite, error) {
	state := struct {
		NetParamsOverwrite *GenesisStateOverwrite `json:"network_parameters_checkpoint_overwrite"`
	}{}
	err := json.Unmarshal(bytes, &state)
	if err != nil {
		return nil, err
	}
	if state.NetParamsOverwrite == nil {
		return nil, nil // not an error, not mandatory to have overwrite list
	}
	return *state.NetParamsOverwrite, nil
}
