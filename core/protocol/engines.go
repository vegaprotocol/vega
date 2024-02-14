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

package protocol

import (
	"code.vegaprotocol.io/vega/core/evtforward"
	evtfwdeth "code.vegaprotocol.io/vega/core/evtforward/ethereum"
	"code.vegaprotocol.io/vega/core/types"
)

type EventForwarderEngine interface {
	ReloadConf(evtforward.Config)
	SetupEthereumEngine(evtfwdeth.Client, evtfwdeth.Forwarder, evtfwdeth.Config, *types.EthereumConfig, evtfwdeth.Assets) error
	SetupSecondaryEthereumEngine(evtfwdeth.Client, evtfwdeth.Forwarder, evtfwdeth.Config, *types.SecondaryEthereumConfig, evtfwdeth.Assets) error
	Start()
	Stop()

	// methods used to update starting blocks of the eef
	UpdateCollateralStartingBlock(uint64)
	UpdateStakingStartingBlock(uint64)
	UpdateMultisigControlStartingBlock(uint64)
}
