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
	FilterCollateralEvents(ctx context.Context, startAt uint64, cb OnEventFound) uint64
	FilterStakingEvents(ctx context.Context, startAt uint64, cb OnEventFound) uint64
	CurrentHeight(context.Context) uint64
}

type Engine struct {
	log    *logging.Logger
	poller *poller

	filterer  Filterer
	forwarder Forwarder

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
	nextCollateralBlockNumber := e.nextCollateralBlockNumber
	lastBlockMatched := e.filterer.FilterCollateralEvents(ctx, nextCollateralBlockNumber, func(event *commandspb.ChainEvent) {
		e.forwarder.ForwardFromSelf(event)
	})
	e.nextCollateralBlockNumber = lastBlockMatched + 1

	nextStakingBlockNumber := e.nextStakingBlockNumber
	lastBlockMatched = e.filterer.FilterStakingEvents(ctx, nextStakingBlockNumber, func(event *commandspb.ChainEvent) {
		e.forwarder.ForwardFromSelf(event)
	})
	e.nextStakingBlockNumber = lastBlockMatched + 1
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
