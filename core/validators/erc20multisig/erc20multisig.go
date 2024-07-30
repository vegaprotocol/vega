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
	scope string,
) *Topology {
	ocv := NewOnChainVerifier(config, log, ethClient, ethConfirmation)
	top := NewTopology(config, log, witness, ocv, broker, scope)

	if scope == "primary" {
		_ = netp.Watch(netparams.WatchParam{
			Param: netparams.BlockchainsPrimaryEthereumConfig,
			Watcher: func(_ context.Context, cfg interface{}) error {
				ethCfg, err := types.EthereumConfigFromUntypedProto(cfg)
				if err != nil {
					return fmt.Errorf("ERC20 multisig didn't receive a valid Ethereum configuration: %w", err)
				}
				ocv.UpdateMultiSigAddress(ethCfg.MultiSigControl().Address(), ethCfg.ChainID())
				top.SetChainID(ethCfg.ChainID())
				return nil
			},
		})
	} else {
		_ = netp.Watch(netparams.WatchParam{
			Param: netparams.BlockchainsEVMBridgeConfigs,
			Watcher: func(_ context.Context, cfg interface{}) error {
				cfgs, err := types.EVMChainConfigFromUntypedProto(cfg)
				if err != nil {
					return fmt.Errorf("ERC20 multisig didn't receive a valid Ethereum configuration: %w", err)
				}
				cfgs.String(log)
				ethCfg := cfgs.Configs[0]
				ocv.UpdateMultiSigAddress(ethCfg.MultiSigControl().Address(), ethCfg.ChainID())
				top.SetChainID(ethCfg.ChainID())
				return nil
			},
		})
	}

	return top
}
