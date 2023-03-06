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

package protocol

import (
	"code.vegaprotocol.io/vega/core/evtforward"
	evtfwdeth "code.vegaprotocol.io/vega/core/evtforward/ethereum"
	"code.vegaprotocol.io/vega/core/types"
)

type EventForwarderEngine interface {
	ReloadConf(evtforward.Config)
	SetupEthereumEngine(evtfwdeth.Client, evtfwdeth.Forwarder, evtfwdeth.Config, *types.EthereumConfig, evtfwdeth.Assets) error
	Start()
	Stop()

	// methods used to update starting blocks of the eef
	UpdateCollateralStartingBlock(uint64)
	UpdateStakingStartingBlock(uint64)
	UpdateMultisigControlStartingBlock(uint64)
}
