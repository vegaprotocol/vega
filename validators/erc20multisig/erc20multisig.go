package erc20multisig

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/vega/broker"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/netparams"
	"code.vegaprotocol.io/vega/types"
)

func NewERC20MultisigTopology(
	config Config,
	log *logging.Logger,
	witness Witness,
	broker broker.BrokerI,
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
