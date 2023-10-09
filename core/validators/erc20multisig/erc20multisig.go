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

package erc20multisig

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/vega/core/broker"
	"code.vegaprotocol.io/vega/core/netparams"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/logging"
)

func NewERC20MultisigTopology(
	config Config,
	log *logging.Logger,
	witness Witness,
	broker broker.Interface,
	ethClient EthereumClient,
	ethConfirmation EthConfirmations,
	netp *netparams.Store,
) *Topology {
	ocv := NewOnChainVerifier(config, log, ethClient, ethConfirmation)
	_ = netp.Watch(netparams.WatchParam{
		Param: netparams.BlockchainsEthereumConfig,
		Watcher: func(_ context.Context, cfg interface{}) error {
			ethCfg, err := types.EthereumConfigFromUntypedProto(cfg)
			if err != nil {
				return fmt.Errorf("staking didn't receive a valid Ethereum configuration: %w", err)
			}

			ocv.UpdateMultiSigAddress(ethCfg.MultiSigControl().Address())
			return nil
		},
	})

	return NewTopology(config, log, witness, ocv, broker)
}
