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
