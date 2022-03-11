package staking

import (
	"context"
	"fmt"
	"math/big"

	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/netparams"
	"code.vegaprotocol.io/vega/types"

	ethtypes "github.com/ethereum/go-ethereum/core/types"
)

type AllEthereumClient interface {
	EthereumClient
	EthereumClientConfirmations
	EthereumClientCaller
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/ethereum_client_confirmations_mock.go -package mocks code.vegaprotocol.io/vega/staking EthereumClientConfirmations
type EthereumClientConfirmations interface {
	HeaderByNumber(context.Context, *big.Int) (*ethtypes.Header, error)
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/ethereum_event_source_mock.go -package mocks code.vegaprotocol.io/vega/staking EthereumEventSource
type EthereumEventSource interface {
	UpdateStakingStartingBlock(uint64)
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
	ethCfns EthConfirmations,
	ethEventSource EthereumEventSource,
) (*Accounting, *StakeVerifier, *Checkpoint) {
	log = log.Named(namedLogger)
	log.SetLevel(cfg.Level.Get())
	accs := NewAccounting(log, cfg, broker, ethClient, evtFwd, witness, tt, isValidator)
	ocv := NewOnChainVerifier(cfg, log, ethClient, ethCfns)
	stakeV := NewStakeVerifier(log, cfg, accs, tt, witness, broker, ocv)

	_ = netp.Watch(netparams.WatchParam{
		Param: netparams.BlockchainsEthereumConfig,
		Watcher: func(_ context.Context, cfg interface{}) error {
			ethCfg, err := types.EthereumConfigFromUntypedProto(cfg)
			if err != nil {
				return fmt.Errorf("staking didn't receive a valid Ethereum configuration: %w", err)
			}

			ocv.UpdateStakingBridgeAddresses(ethCfg.StakingBridgeAddresses())

			// We just need one of the staking bridges.
			if err := accs.UpdateStakingBridgeAddress(ethCfg.StakingBridgeAddresses()[0]); err != nil {
				return fmt.Errorf("couldn't update Ethereum configuration in accounting: %w", err)
			}

			return nil
		},
	})

	return accs, stakeV, NewCheckpoint(log, accs, stakeV, ethEventSource)
}
