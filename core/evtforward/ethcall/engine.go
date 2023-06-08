package ethcall

import (
	"context"
	"fmt"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/core/types"

	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/protos/vega"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	"github.com/ethereum/go-ethereum"
)

// Still TODO
//   - on tick check every block since last tick not just current
//   - submit some sort of error event if call fails
//   - know when datasources stop being active and remove them
//     -- because e.g. market is dead, or amended to have different source
//     -- or because trigger will never fire again
//   - what to do about catching up e.g. if node is restarted
type EthReaderCaller interface {
	ethereum.ContractCaller
	ethereum.ChainReader
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/forwarder_mock.go -package mocks code.vegaprotocol.io/vega/core/evtforward/ethcall Forwarder
type Forwarder interface {
	ForwardFromSelf(*commandspb.ChainEvent)
}

type blockish interface {
	NumberU64() uint64
	Time() uint64
}

type Engine struct {
	log                   *logging.Logger
	cfg                   Config
	client                EthReaderCaller
	dataSources           map[string]DataSource
	forwarder             Forwarder
	prevBlock             blockish
	cancelEthereumQueries context.CancelFunc
	poller                *poller
	mu                    sync.Mutex
}

func NewEngine(log *logging.Logger, cfg Config, client EthReaderCaller, forwarder Forwarder) *Engine {
	e := &Engine{
		log:         log,
		cfg:         cfg,
		client:      client,
		forwarder:   forwarder,
		dataSources: make(map[string]DataSource),
		poller:      newPoller(cfg.PollEvery.Get()),
	}

	go e.Start()
	return e
}

// Start starts the polling of the Ethereum bridges, listens to the events
// they emit and forward it to the network.
func (e *Engine) Start() {
	ctx, cancelEthereumQueries := context.WithCancel(context.Background())
	defer cancelEthereumQueries()

	e.cancelEthereumQueries = cancelEthereumQueries

	if e.log.IsDebug() {
		e.log.Debug("Starting ethereum contract call polling engine")
	}

	e.poller.Loop(func() {
		e.OnTick(ctx, time.Now())
	})
}

func (e *Engine) Stop() {
	// Notify to stop on next iteration.
	e.poller.Stop()
	// Cancel any ongoing queries against Ethereum.
	if e.cancelEthereumQueries != nil {
		e.cancelEthereumQueries()
	}
}

func (e *Engine) GetDataSource(id string) (DataSource, bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if source, ok := e.dataSources[id]; ok {
		return source, true
	}

	return DataSource{}, false
}

func (e *Engine) OnSpecActivated(ctx context.Context, spec types.OracleSpec) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	switch d := spec.ExternalDataSourceSpec.Spec.Data.Content().(type) {
	case types.EthCallSpec:
		id := spec.ExternalDataSourceSpec.Spec.ID
		if _, ok := e.dataSources[id]; ok {
			return fmt.Errorf("duplicate spec: %s", id)
		}

		dataSource, err := NewDataSource(d)
		if err != nil {
			return fmt.Errorf("failed to create data source: %w", err)
		}

		e.dataSources[id] = dataSource
	}

	return nil
}

func (e *Engine) OnSpecDeactivated(ctx context.Context, spec types.OracleSpec) {
	e.mu.Lock()
	defer e.mu.Unlock()
	switch spec.ExternalDataSourceSpec.Spec.Data.Content().(type) {
	case *types.EthCallSpec:
		id := spec.ExternalDataSourceSpec.Spec.ID
		delete(e.dataSources, id)
	}
}

func (e *Engine) OnTick(ctx context.Context, wallTime time.Time) {
	//TODO: maybe don't want to hold this lock all the time as eth call could be slow
	e.mu.Lock()
	defer e.mu.Unlock()
	block, err := e.client.BlockByNumber(ctx, nil)
	if err != nil {
		e.log.Errorf("failed to get current block header: %w", err)
		return
	}

	e.log.Info("tick",
		logging.Time("vegaTime", wallTime),
		logging.BigInt("ethBlock", block.Number()),
		logging.Time("ethTime", time.Unix(int64(block.Time()), 0)))

	if e.prevBlock == nil {
		e.prevBlock = block
		return
	}

	for specID, datasource := range e.dataSources {
		if datasource.Trigger(e.prevBlock, block) {
			res, err := datasource.Call.Call(ctx, e.client, block.Number())
			if err != nil {
				e.log.Errorf("failed to call contract: %w", err)
				continue
			}
			event := makeChainEvent(res, specID, block)
			e.forwarder.ForwardFromSelf(event)
		}
	}
	e.prevBlock = block
}

func makeChainEvent(res Result, specID string, block blockish) *commandspb.ChainEvent {
	ce := commandspb.ChainEvent{
		TxId:  "", // NA
		Nonce: 0,  // NA
		Event: &commandspb.ChainEvent_ContractCall{
			ContractCall: &vega.EthContractCallEvent{
				SpecId:      specID,
				BlockHeight: block.NumberU64(),
				BlockTime:   block.Time(),
				Result:      res.Bytes,
			},
		},
	}

	return &ce
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

// This is copy-pasted from the ethereum engine; at some point this two should probably be folded into one,
// but just for now keep them separate to ensure we don't break existing functionality.
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
