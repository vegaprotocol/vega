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

package genesis

import (
	"code.vegaprotocol.io/vega/core/assets"
	"code.vegaprotocol.io/vega/core/blockchain/abci"
	"code.vegaprotocol.io/vega/core/checkpoint"
	"code.vegaprotocol.io/vega/core/limits"
	"code.vegaprotocol.io/vega/core/netparams"
	"code.vegaprotocol.io/vega/core/validators"
)

type State struct {
	Assets             assets.GenesisState             `json:"assets"`
	Validators         validators.GenesisState         `json:"validators"`
	Network            abci.GenesisState               `json:"network"`
	NetParams          netparams.GenesisState          `json:"network_parameters"`
	NetParamsOverwrite netparams.GenesisStateOverwrite `json:"network_parameters_checkpoint_overwrite"`
	Limits             limits.GenesisState             `json:"network_limits"`
	Checkpoint         checkpoint.GenesisState         `json:"checkpoint"`
}

func DefaultState() State {
	return State{
		Limits:     limits.DefaultGenesisState(),
		Assets:     assets.DefaultGenesisState(),
		Validators: validators.DefaultGenesisState(),
		Network:    abci.DefaultGenesis(),
		NetParams:  netparams.DefaultGenesisState(),
		Checkpoint: checkpoint.DefaultGenesisState(),
	}
}
