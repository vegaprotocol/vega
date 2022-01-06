package ethereum

import (
	"context"
	"time"

	commandspb "code.vegaprotocol.io/protos/vega/commands/v1"
	"code.vegaprotocol.io/vega/logging"
)

const (
	engineLogger            = "engine"
	durationBetweenTwoRetry = 15 * time.Second
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/forwarder_mock.go -package mocks code.vegaprotocol.io/vega/evtforward/ethereum Forwarder
type Forwarder interface {
	ForwardFromSelf(*commandspb.ChainEvent)
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/filterer_mock.go -package mocks code.vegaprotocol.io/vega/evtforward/ethereum Filterer
type Filterer interface {
	AssetWithdrawnEvents(ctx context.Context, startAt uint64, cb OnEventFound) uint64
	AssetDepositedEvents(ctx context.Context, startAt uint64, cb OnEventFound) uint64
	AssetListEvents(ctx context.Context, startAt uint64, cb OnEventFound) uint64
	AssetDelistEvents(ctx context.Context, startAt uint64, cb OnEventFound) uint64
	StakeDepositedEvents(ctx context.Context, startAt uint64, cb OnEventFound) uint64
	StakeRemovedEvents(ctx context.Context, startAt uint64, cb OnEventFound) uint64
	CurrentHeight(context.Context) uint64
}

type Engine struct {
	log    *logging.Logger
	poller *poller

	filterer  Filterer
	forwarder Forwarder

	// We save the smallest block number of the last matched events because the
	// overall filtering is synchronous, each type being filtered independently.
	// This means the first filtering step might have stopped earlier than the
	// last step. Starting from the highest block number would make the
	// processing skip blocks for the first type being filtered.
	//
	// Example:
	//
	// - S:  Starting block number for filtering for a given type
	// - Ln: Last matched event block number for a given type.
	// - Hn: Block number at which the filtering stopped, that should match the
	// 		 current height of Ethereum.
	//
	// Withdraw   S----L1---H1
	// Deposit    S-------------L2-----H2
	//
	// While processing the Deposit, the current height (Hn) of Ethereum might
	// have move forward, reaching H2. So, when re-filtering Withdraw events,
	// we don't want to skip the events between H1 and H2. As a result, we could
	// save H1, however, due to some API limitations, we don't know about H1, we
	// only know about L1, so we have to save L1, meaning the block number of
	// the last matched event from the first step, only.
	nextCollateralBlockNumber uint64
	nextStakingBlockNumber    uint64

	cancelEthereumQueries context.CancelFunc
}

func NewEngine(
	log *logging.Logger,
	filterer Filterer,
	forwarder Forwarder,
	stakingDeploymentBlockHeight uint64,
) *Engine {
	l := log.Named(engineLogger)

	return &Engine{
		log:       l,
		poller:    newPoller(),
		filterer:  filterer,
		forwarder: forwarder,

		// Setting up the starting block number for event filtering on the
		// staking bridge.
		nextStakingBlockNumber: stakingDeploymentBlockHeight,
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

	// Setting the starting block number for event gathering on the collateral
	// bridge.
	currentHeight := e.filterer.CurrentHeight(ctx)
	e.nextCollateralBlockNumber = currentHeight

	e.poller.Loop(func() {
		e.gatherEvents(ctx)
	})
}

func (e *Engine) gatherEvents(ctx context.Context) {
	// Collateral bridge
	nextCollateralBlockNumber := e.nextCollateralBlockNumber
	lastBlockMatched := e.filterer.AssetWithdrawnEvents(ctx, nextCollateralBlockNumber, func(event *commandspb.ChainEvent) {
		e.forwarder.ForwardFromSelf(event)
	})
	// We update nextCollateralBlockNumber with lastBlockMatched coming from the
	// first step of the collateral bridge filtering only.
	// More details in the property comment.
	e.nextCollateralBlockNumber = lastBlockMatched + 1

	_ = e.filterer.AssetDepositedEvents(ctx, nextCollateralBlockNumber, func(event *commandspb.ChainEvent) {
		e.forwarder.ForwardFromSelf(event)
	})

	_ = e.filterer.AssetListEvents(ctx, nextCollateralBlockNumber, func(event *commandspb.ChainEvent) {
		e.forwarder.ForwardFromSelf(event)
	})

	_ = e.filterer.AssetDelistEvents(ctx, nextCollateralBlockNumber, func(event *commandspb.ChainEvent) {
		e.forwarder.ForwardFromSelf(event)
	})

	// Staking bridge
	nextStakingBlockNumber := e.nextStakingBlockNumber
	lastBlockMatched = e.filterer.StakeDepositedEvents(ctx, nextStakingBlockNumber, func(event *commandspb.ChainEvent) {
		e.forwarder.ForwardFromSelf(event)
	})
	// We update nextStakingBlockNumber with lastBlockMatched coming from the
	// first step of the staking bridge filtering only.
	// More details in the property comment.
	e.nextStakingBlockNumber = lastBlockMatched + 1

	_ = e.filterer.StakeRemovedEvents(ctx, nextStakingBlockNumber, func(event *commandspb.ChainEvent) {
		e.forwarder.ForwardFromSelf(event)
	})
}

// Stop stops the engine, its polling and event forwarding.
func (e *Engine) Stop() {
	// Notify to stop on next iteration.
	e.poller.Stop()
	// Cancel any ongoing queries against Ethereum.
	e.cancelEthereumQueries()
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
