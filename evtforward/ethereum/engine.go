package ethereum

import (
	"context"
	"time"

	commandspb "code.vegaprotocol.io/protos/vega/commands/v1"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/types"
)

const (
	engineLogger            = "engine"
	durationBetweenTwoRetry = 20 * time.Second
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/forwarder_mock.go -package mocks code.vegaprotocol.io/vega/evtforward/ethereum Forwarder
type Forwarder interface {
	ForwardFromSelf(*commandspb.ChainEvent)
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/filterer_mock.go -package mocks code.vegaprotocol.io/vega/evtforward/ethereum Filterer
type Filterer interface {
	FilterCollateralEvents(ctx context.Context, startAt, stopAt uint64, cb OnEventFound)
	FilterStakingEvents(ctx context.Context, startAt, stopAt uint64, cb OnEventFound)
	FilterVestingEvents(ctx context.Context, startAt, stopAt uint64, cb OnEventFound)
	FilterMultiSigControlEvents(ctx context.Context, startAt, stopAt uint64, cb OnEventFound)
	CurrentHeight(context.Context) uint64
}

type Engine struct {
	log    *logging.Logger
	poller *poller

	filterer  Filterer
	forwarder Forwarder

	nextCollateralBlockNumber      uint64
	nextMultiSigControlBlockNumber uint64
	nextStakingBlockNumber         uint64
	nextVestingBlockNumber         uint64

	shouldFilterVestingBridge bool
	shouldFilterStakingBridge bool

	cancelEthereumQueries context.CancelFunc
}

func NewEngine(
	log *logging.Logger,
	filterer Filterer,
	forwarder Forwarder,
	stakingDeployment types.EthereumContract,
	vestingDeployment types.EthereumContract,
	multiSigDeployment types.EthereumContract,
) *Engine {
	l := log.Named(engineLogger)

	return &Engine{
		log:                            l,
		poller:                         newPoller(),
		filterer:                       filterer,
		forwarder:                      forwarder,
		shouldFilterStakingBridge:      stakingDeployment.HasAddress(),
		nextStakingBlockNumber:         stakingDeployment.DeploymentBlockHeight(),
		shouldFilterVestingBridge:      vestingDeployment.HasAddress(),
		nextVestingBlockNumber:         vestingDeployment.DeploymentBlockHeight(),
		nextMultiSigControlBlockNumber: multiSigDeployment.DeploymentBlockHeight(),
	}
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

	e.nextCollateralBlockNumber = e.filterer.CurrentHeight(ctx)

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

func (e *Engine) gatherEvents(ctx context.Context) {
	currentHeight := e.filterer.CurrentHeight(ctx)

	// // Ensure we are not issuing a filtering request for non-existing block.
	// if e.nextCollateralBlockNumber <= currentHeight {
	// 	e.filterer.FilterCollateralEvents(ctx, e.nextCollateralBlockNumber, currentHeight, func(event *commandspb.ChainEvent) {
	// 		e.forwarder.ForwardFromSelf(event)
	// 	})
	// 	e.nextCollateralBlockNumber = currentHeight + 1
	// }

	// // Ensure we are not issuing a filtering request for non-existing block.
	// if e.shouldFilterStakingBridge && e.nextStakingBlockNumber <= currentHeight {
	// 	e.filterer.FilterStakingEvents(ctx, e.nextStakingBlockNumber, currentHeight, func(event *commandspb.ChainEvent) {
	// 		e.forwarder.ForwardFromSelf(event)
	// 	})
	// 	e.nextStakingBlockNumber = currentHeight + 1
	// }

	// // Ensure we are not issuing a filtering request for non-existing block.
	// if e.shouldFilterVestingBridge && e.nextVestingBlockNumber <= currentHeight {
	// 	e.filterer.FilterVestingEvents(ctx, e.nextVestingBlockNumber, currentHeight, func(event *commandspb.ChainEvent) {
	// 		e.forwarder.ForwardFromSelf(event)
	// 	})
	// 	e.nextVestingBlockNumber = currentHeight + 1
	// }

	// Ensure we are not issuing a filtering request for non-existing block.
	if e.nextMultiSigControlBlockNumber <= currentHeight {
		e.filterer.FilterMultiSigControlEvents(ctx, e.nextMultiSigControlBlockNumber, currentHeight, func(event *commandspb.ChainEvent) {
			e.forwarder.ForwardFromSelf(event)
		})
		e.nextMultiSigControlBlockNumber = currentHeight + 1
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
	ticker *time.Ticker
	done   chan bool
}

func newPoller() *poller {
	return &poller{
		ticker: time.NewTicker(durationBetweenTwoRetry),
		done:   make(chan bool, 1),
	}
}

// Loop starts the poller loop until it's broken, using the Stop method.
func (s *poller) Loop(fn func()) {
	defer func() {
		s.ticker.Stop()
		s.ticker.Reset(durationBetweenTwoRetry)
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
