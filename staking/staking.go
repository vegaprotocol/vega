package staking

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/netparams"
	"code.vegaprotocol.io/vega/types"
)

type AllEthereumClient interface {
	EthereumClient
	EthereumClientConfirmations
	EthereumClientCaller
}

func New(
	log *logging.Logger,
	cfg Config,
	broker Broker,
	tt TimeTicker,
	witness Witness,
	ethClient AllEthereumClient,
	netp *netparams.Store,
	evtFwd EvtForwarder,
	isValidator bool,
) (*Accounting, *StakeVerifier) {
	log = log.Named(namedLogger)
	log.SetLevel(cfg.Level.Get())
	accs := NewAccounting(log, cfg, broker, ethClient, evtFwd, witness, tt, isValidator)
	ethCfns := NewEthereumConfirmations(ethClient, nil)
	ocv := NewOnChainVerifier(cfg, log, ethClient, ethCfns)
	stakeV := NewStakeVerifier(log, cfg, accs, tt, witness, broker, ocv)

	_ = netp.Watch(netparams.WatchParam{
		Param: netparams.BlockchainsEthereumConfig,
		Watcher: func(_ context.Context, cfg interface{}) error {
			ethCfg, err := types.EthereumConfigFromUntypedProto(cfg)
			if err != nil {
				return fmt.Errorf("staking didn't receive a valid Ethereum configuration: %w", err)
			}

			ethCfns.UpdateConfirmations(ethCfg.Confirmations())
			ocv.UpdateStakingBridgeAddresses(ethCfg.StakingBridgeAddresses())

			// We just need one of the staking bridges.
			if err := accs.UpdateStakingBridgeAddress(ethCfg.StakingBridgeAddresses()[0]); err != nil {
				return fmt.Errorf("couldn't update Ethereum configuration in accounting: %w", err)
			}

			return nil
		},
	})

	return accs, stakeV
}
