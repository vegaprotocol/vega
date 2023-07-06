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
		if _, ok := Deprecated[k]; ok {
			continue
		}
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
