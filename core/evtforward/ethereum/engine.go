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
	"errors"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/protos/vega"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
)

const (
	engineLogger = "engine"
)

var ErrInvalidHeartbeat = errors.New("forwarded heartbeat is invalid")

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
	GetEthTime(ctx context.Context, atBlock uint64) (uint64, error)
}

// Contract wrapper around EthereumContract to keep track of the block heights we've checked.
type Contract struct {
	types.EthereumContract
	next uint64 // the block height we will next check for events, all block heights less than this will have events sent in
	last uint64 // the block height we last sent out an event for this contract, including heartbeats
}

type Engine struct {
	cfg    Config
	log    *logging.Logger
	poller *poller

	filterer  Filterer
	forwarder Forwarder

	chainID string

	stakingDeployment    *Contract
	vestingDeployment    *Contract
	collateralDeployment *Contract
	multisigDeployment   *Contract
	mu                   sync.Mutex

	cancelEthereumQueries context.CancelFunc

	// the number of blocks between heartbeats
	heartbeatInterval uint64
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
	collateralDeployment types.EthereumContract,
	chainID string,
	blockTime time.Duration,
) *Engine {
	l := log.Named(engineLogger)

	// given that the EVM bridge configs are and array the "unset" values do not get populated
	// with reasonable defaults so we need to make sure they are set to something reasonable
	// if they are left out
	cfg.setDefaults()

	// calculate the number of blocks in an hour, this will be the interval we send out heartbeats
	heartbeatTime := cfg.HeartbeatIntervalForTestOnlyDoNotChange.Duration
	heartbeatInterval := heartbeatTime.Seconds() / blockTime.Seconds()

	return &Engine{
		cfg:                  cfg,
		log:                  l,
		poller:               newPoller(cfg.PollEventRetryDuration.Get()),
		filterer:             filterer,
		forwarder:            fwdWrapper{forwarder, chainID},
		stakingDeployment:    &Contract{stakingDeployment, stakingDeployment.DeploymentBlockHeight(), stakingDeployment.DeploymentBlockHeight()},
		vestingDeployment:    &Contract{vestingDeployment, vestingDeployment.DeploymentBlockHeight(), vestingDeployment.DeploymentBlockHeight()},
		multisigDeployment:   &Contract{multiSigDeployment, multiSigDeployment.DeploymentBlockHeight(), multiSigDeployment.DeploymentBlockHeight()},
		collateralDeployment: &Contract{collateralDeployment, collateralDeployment.DeploymentBlockHeight(), collateralDeployment.DeploymentBlockHeight()},
		chainID:              chainID,
		heartbeatInterval:    uint64(heartbeatInterval),
	}
}

func (e *Engine) UpdateCollateralStartingBlock(b uint64) {
	e.collateralDeployment.next = b
}

func (e *Engine) UpdateStakingStartingBlock(b uint64) {
	e.vestingDeployment.next = b
	e.stakingDeployment.next = b
}

func (e *Engine) UpdateMultiSigControlStartingBlock(b uint64) {
	e.multisigDeployment.next = b
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
				logging.String("chain-id", e.chainID),
				logging.Uint64("next-collateral-block-number", e.collateralDeployment.next),
				logging.Uint64("next-multisig-control-block-number", e.multisigDeployment.next),
				logging.Uint64("next-staking-block-number", e.stakingDeployment.next),
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
	e.mu.Lock()
	defer e.mu.Unlock()

	// Ensure we are not issuing a filtering request for non-existing block.
	if ok, nextHeight := issueFilteringRequest(e.collateralDeployment.next, currentHeight, nBlocks); ok {
		e.filterer.FilterCollateralEvents(ctx, e.collateralDeployment.next, nextHeight, func(event *commandspb.ChainEvent, h uint64) {
			e.forwarder.ForwardFromSelf(event)
			e.collateralDeployment.last = h
		})
		e.collateralDeployment.next = nextHeight + 1
		e.sendHeartbeat(e.collateralDeployment)
	}

	// Ensure we are not issuing a filtering request for non-existing block.
	if e.stakingDeployment.HasAddress() {
		if ok, nextHeight := issueFilteringRequest(e.stakingDeployment.next, currentHeight, nBlocks); ok {
			e.filterer.FilterStakingEvents(ctx, e.stakingDeployment.next, nextHeight, func(event *commandspb.ChainEvent, h uint64) {
				e.forwarder.ForwardFromSelf(event)
				e.stakingDeployment.last = h
			})
			e.stakingDeployment.next = nextHeight + 1
			e.sendHeartbeat(e.stakingDeployment)
		}
	}

	// Ensure we are not issuing a filtering request for non-existing block.
	if e.vestingDeployment.HasAddress() {
		if ok, nextHeight := issueFilteringRequest(e.vestingDeployment.next, currentHeight, nBlocks); ok {
			e.filterer.FilterVestingEvents(ctx, e.vestingDeployment.next, nextHeight, func(event *commandspb.ChainEvent, h uint64) {
				e.forwarder.ForwardFromSelf(event)
				e.vestingDeployment.last = h
			})
			e.vestingDeployment.next = nextHeight + 1
			e.sendHeartbeat(e.vestingDeployment)
		}
	}

	// Ensure we are not issuing a filtering request for non-existing block.
	if ok, nextHeight := issueFilteringRequest(e.multisigDeployment.next, currentHeight, nBlocks); ok {
		e.filterer.FilterMultisigControlEvents(ctx, e.multisigDeployment.next, nextHeight, func(event *commandspb.ChainEvent, h uint64) {
			e.forwarder.ForwardFromSelf(event)
			e.multisigDeployment.last = h
		})
		e.multisigDeployment.next = nextHeight + 1
		e.sendHeartbeat(e.multisigDeployment)
	}
}

// sendHeartbeat checks whether it has been more than and hour since the validator sent a chain event for the given contract
// and if it has will send a heartbeat chain event so that core has an recent view on the last block checked for new events.
func (e *Engine) sendHeartbeat(contract *Contract) {
	// how many heartbeat intervals between the last sent event, and the block height we're checking next
	n := (contract.next - contract.last) / e.heartbeatInterval
	if n == 0 {
		return
	}

	height := contract.last + n*e.heartbeatInterval
	time, err := e.filterer.GetEthTime(context.Background(), height)
	if err != nil {
		e.log.Error("unable to find eth-time for contract heartbeat",
			logging.Uint64("height", height),
			logging.String("chain-id", e.chainID),
			logging.Error(err),
		)
		return
	}

	e.forwarder.ForwardFromSelf(
		&commandspb.ChainEvent{
			TxId:  "internal", // NA
			Nonce: 0,          // NA
			Event: &commandspb.ChainEvent_Heartbeat{
				Heartbeat: &vega.ERC20Heartbeat{
					ContractAddress: contract.HexAddress(),
					BlockHeight:     height,
					SourceChainId:   e.chainID,
					BlockTime:       time,
				},
			},
		},
	)
	contract.last = height
}

// VerifyHeart checks that the block height of the heartbeat exists and contains the correct block time. It also
// checks that this node has checked the logs of the given contract address up to at least the given height.
func (e *Engine) VerifyHeartbeat(ctx context.Context, height uint64, chainID string, address string, blockTime uint64) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	t, err := e.filterer.GetEthTime(ctx, height)
	if err != nil {
		return err
	}

	if t != blockTime {
		return ErrInvalidHeartbeat
	}

	var lastChecked uint64
	if e.collateralDeployment.HexAddress() == address {
		lastChecked = e.collateralDeployment.next - 1
	}

	if e.multisigDeployment.HexAddress() == address {
		lastChecked = e.multisigDeployment.next - 1
	}

	if e.stakingDeployment.HexAddress() == address {
		lastChecked = e.stakingDeployment.next - 1
	}

	if e.vestingDeployment.HexAddress() == address {
		lastChecked = e.vestingDeployment.next - 1
	}

	// if the heartbeat block height is higher than the last block *this* node has checked for logs
	// on the contract, then fail the verification
	if lastChecked < height {
		return ErrInvalidHeartbeat
	}
	return nil
}

// UpdateStartingBlock sets the height that we should starting looking for new events from for the given bridge contract address.
func (e *Engine) UpdateStartingBlock(address string, block uint64) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if block == 0 {
		return
	}

	if e.collateralDeployment.HexAddress() == address {
		e.collateralDeployment.last = block
		e.collateralDeployment.next = block
		return
	}

	if e.multisigDeployment.HexAddress() == address {
		e.multisigDeployment.last = block
		e.multisigDeployment.next = block
		return
	}

	if e.stakingDeployment.HexAddress() == address {
		e.stakingDeployment.last = block
		e.stakingDeployment.next = block
		return
	}

	if e.vestingDeployment.HexAddress() == address {
		e.vestingDeployment.last = block
		e.vestingDeployment.next = block
		return
	}

	e.log.Warn("unexpected contract address starting block",
		logging.String("chain-id", e.chainID),
		logging.String("contract-address", address),
	)
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
