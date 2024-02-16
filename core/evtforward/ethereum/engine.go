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

package ethereum

import (
	"context"
	"time"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/logging"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
)

const (
	engineLogger = "engine"
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/forwarder_mock.go -package mocks code.vegaprotocol.io/vega/core/evtforward/ethereum Forwarder
type Forwarder interface {
	ForwardFromSelf(*commandspb.ChainEvent)
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/filterer_mock.go -package mocks code.vegaprotocol.io/vega/core/evtforward/ethereum Filterer
type Filterer interface {
	FilterCollateralEvents(ctx context.Context, startAt, stopAt uint64, cb OnEventFound)
	FilterStakingEvents(ctx context.Context, startAt, stopAt uint64, cb OnEventFound)
	FilterVestingEvents(ctx context.Context, startAt, stopAt uint64, cb OnEventFound)
	FilterMultisigControlEvents(ctx context.Context, startAt, stopAt uint64, cb OnEventFound)
	CurrentHeight(context.Context) uint64
}

type Engine struct {
	cfg    Config
	log    *logging.Logger
	poller *poller

	filterer  Filterer
	forwarder Forwarder

	nextCollateralBlockNumber      uint64
	nextMultiSigControlBlockNumber uint64
	nextStakingBlockNumber         uint64
	nextVestingBlockNumber         uint64

	chainID string

	shouldFilterVestingBridge bool
	shouldFilterStakingBridge bool

	cancelEthereumQueries context.CancelFunc
}

type fwdWrapper struct {
	f       Forwarder
	chainID string
}

func (f fwdWrapper) ForwardFromSelf(event *commandspb.ChainEvent) {
	// add the chainID of the source on events where this is necessary
	switch ev := event.Event.(type) {
	case *commandspb.ChainEvent_Erc20:
		ev.Erc20.ChainId = f.chainID
	case *commandspb.ChainEvent_Erc20Multisig:
		ev.Erc20Multisig.ChainId = f.chainID
	default:
		// do nothing
	}

	f.f.ForwardFromSelf(event)
}

func NewEngine(
	cfg Config,
	log *logging.Logger,
	filterer Filterer,
	forwarder Forwarder,
	stakingDeployment types.EthereumContract,
	vestingDeployment types.EthereumContract,
	multiSigDeployment types.EthereumContract,
	chainID string,
) *Engine {
	l := log.Named(engineLogger)

	return &Engine{
		cfg:                            cfg,
		log:                            l,
		poller:                         newPoller(cfg.PollEventRetryDuration.Get()),
		filterer:                       filterer,
		forwarder:                      fwdWrapper{forwarder, chainID},
		shouldFilterStakingBridge:      stakingDeployment.HasAddress(),
		nextStakingBlockNumber:         stakingDeployment.DeploymentBlockHeight(),
		shouldFilterVestingBridge:      vestingDeployment.HasAddress(),
		nextVestingBlockNumber:         vestingDeployment.DeploymentBlockHeight(),
		nextMultiSigControlBlockNumber: multiSigDeployment.DeploymentBlockHeight(),
		chainID:                        chainID,
	}
}

func (e *Engine) UpdateCollateralStartingBlock(b uint64) {
	e.nextCollateralBlockNumber = b
}

func (e *Engine) UpdateStakingStartingBlock(b uint64) {
	e.nextStakingBlockNumber = b
	e.nextVestingBlockNumber = b
}

func (e *Engine) UpdateMultiSigControlStartingBlock(b uint64) {
	e.nextMultiSigControlBlockNumber = b
}

func (e *Engine) ReloadConf(cfg Config) {
	e.log.Info("Reloading configuration")

	if e.log.GetLevel() != cfg.Level.Get() {
		e.log.Debug("Updating log level",
			logging.String("old", e.log.GetLevel().String()),
			logging.String("new", cfg.Level.String()),
		)
		e.log.SetLevel(cfg.Level.Get())
	}
}

// Start starts the polling of the Ethereum bridges, listens to the events
// they emit and forward it to the network.
func (e *Engine) Start() {
	ctx, cancelEthereumQueries := context.WithCancel(context.Background())
	defer cancelEthereumQueries()

	e.cancelEthereumQueries = cancelEthereumQueries

	if e.log.IsDebug() {
		e.log.Debug("Start listening for Ethereum events from")
	}

	e.poller.Loop(func() {
		if e.log.IsDebug() {
			e.log.Debug("Clock is ticking, gathering Ethereum events",
				logging.Uint64("next-collateral-block-number", e.nextCollateralBlockNumber),
				logging.Uint64("next-multisig-control-block-number", e.nextMultiSigControlBlockNumber),
				logging.Uint64("next-staking-block-number", e.nextStakingBlockNumber),
			)
		}
		e.gatherEvents(ctx)
	})
}

func issueFilteringRequest(from, to, nBlocks uint64) (ok bool, actualTo uint64) {
	if from > to {
		return false, 0
	}
	return true, min(from+nBlocks, to)
}

func min(a, b uint64) uint64 {
	if a < b {
		return a
	}
	return b
}

func (e *Engine) gatherEvents(ctx context.Context) {
	nBlocks := e.cfg.MaxEthereumBlocks
	currentHeight := e.filterer.CurrentHeight(ctx)

	// Ensure we are not issuing a filtering request for non-existing block.
	if ok, nextHeight := issueFilteringRequest(e.nextCollateralBlockNumber, currentHeight, nBlocks); ok {
		e.filterer.FilterCollateralEvents(ctx, e.nextCollateralBlockNumber, nextHeight, func(event *commandspb.ChainEvent) {
			e.forwarder.ForwardFromSelf(event)
		})
		e.nextCollateralBlockNumber = nextHeight + 1
	}

	// Ensure we are not issuing a filtering request for non-existing block.
	if e.shouldFilterStakingBridge {
		if ok, nextHeight := issueFilteringRequest(e.nextStakingBlockNumber, currentHeight, nBlocks); ok {
			e.filterer.FilterStakingEvents(ctx, e.nextStakingBlockNumber, nextHeight, func(event *commandspb.ChainEvent) {
				e.forwarder.ForwardFromSelf(event)
			})
			e.nextStakingBlockNumber = nextHeight + 1
		}
	}

	// Ensure we are not issuing a filtering request for non-existing block.
	if e.shouldFilterVestingBridge {
		if ok, nextHeight := issueFilteringRequest(e.nextVestingBlockNumber, currentHeight, nBlocks); ok {
			e.filterer.FilterVestingEvents(ctx, e.nextVestingBlockNumber, nextHeight, func(event *commandspb.ChainEvent) {
				e.forwarder.ForwardFromSelf(event)
			})
			e.nextVestingBlockNumber = nextHeight + 1
		}
	}

	// Ensure we are not issuing a filtering request for non-existing block.
	if ok, nextHeight := issueFilteringRequest(e.nextMultiSigControlBlockNumber, currentHeight, nBlocks); ok {
		e.filterer.FilterMultisigControlEvents(ctx, e.nextMultiSigControlBlockNumber, nextHeight, func(event *commandspb.ChainEvent) {
			e.forwarder.ForwardFromSelf(event)
		})
		e.nextMultiSigControlBlockNumber = nextHeight + 1
	}
}

// Stop stops the engine, its polling and event forwarding.
func (e *Engine) Stop() {
	// Notify to stop on next iteration.
	e.poller.Stop()
	// Cancel any ongoing queries against Ethereum.
	if e.cancelEthereumQueries != nil {
		e.cancelEthereumQueries()
	}
}

// poller wraps a poller that ticks every durationBetweenTwoEventFiltering.
type poller struct {
	ticker                  *time.Ticker
	done                    chan bool
	durationBetweenTwoRetry time.Duration
}

func newPoller(durationBetweenTwoRetry time.Duration) *poller {
	return &poller{
		ticker:                  time.NewTicker(durationBetweenTwoRetry),
		done:                    make(chan bool, 1),
		durationBetweenTwoRetry: durationBetweenTwoRetry,
	}
}

// Loop starts the poller loop until it's broken, using the Stop method.
func (s *poller) Loop(fn func()) {
	defer func() {
		s.ticker.Stop()
		s.ticker.Reset(s.durationBetweenTwoRetry)
	}()

	for {
		select {
		case <-s.done:
			return
		case <-s.ticker.C:
			fn()
		}
	}
}

// Stop stops the poller loop.
func (s *poller) Stop() {
	s.done <- true
}
