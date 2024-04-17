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

package staking

import (
	"context"
	"fmt"
	"math/big"

	"code.vegaprotocol.io/vega/core/netparams"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/logging"

	ethtypes "github.com/ethereum/go-ethereum/core/types"
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/mocks.go -package mocks code.vegaprotocol.io/vega/core/staking EvtForwarder,EthereumClientConfirmations,EthereumEventSource,TimeService,EthConfirmations,EthOnChainVerifier,Witness

type AllEthereumClient interface {
	EthereumClient
	EthereumClientConfirmations
	EthereumClientCaller
}

type EthereumClientConfirmations interface {
	HeaderByNumber(context.Context, *big.Int) (*ethtypes.Header, error)
}

type EthereumEventSource interface {
	UpdateStakingStartingBlock(uint64)
}

func New(
	log *logging.Logger,
	cfg Config,
	ts TimeService,
	broker Broker,
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
	accs := NewAccounting(log, cfg, ts, broker, ethClient, evtFwd, witness, isValidator)
	ocv := NewOnChainVerifier(cfg, log, ethClient, ethCfns)
	stakeV := NewStakeVerifier(log, cfg, accs, witness, ts, broker, ocv, ethEventSource)

	_ = netp.Watch(netparams.WatchParam{
		Param: netparams.BlockchainsPrimaryEthereumConfig,
		Watcher: func(_ context.Context, cfg interface{}) error {
			ethCfg, err := types.EthereumConfigFromUntypedProto(cfg)
			if err != nil {
				return fmt.Errorf("staking didn't receive a valid Ethereum configuration: %w", err)
			}

			ocv.UpdateStakingBridgeAddresses(ethCfg.StakingBridgeAddresses())

			// We just need one of the staking bridges.
			if err := accs.UpdateStakingBridgeAddress(ethCfg); err != nil {
				return fmt.Errorf("couldn't update Ethereum configuration in accounting: %w", err)
			}

			return nil
		},
	})

	return accs, stakeV, NewCheckpoint(log, accs, stakeV, ethEventSource)
}
