package ethcall

import (
	"context"
	"fmt"
	"math/big"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/core/types"

	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/protos/vega"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	"github.com/ethereum/go-ethereum"
)

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

type blockIndex struct {
	number uint64
	time   uint64
}

func (b blockIndex) NumberU64() uint64 {
	return b.number
}

func (b blockIndex) Time() uint64 {
	return b.time
}

type Engine struct {
	log                   *logging.Logger
	cfg                   Config
	client                EthReaderCaller
	calls                 map[string]Call
	forwarder             Forwarder
	prevEthBlock          blockish
	cancelEthereumQueries context.CancelFunc
	poller                *poller
	mu                    sync.Mutex
}

func NewEngine(log *logging.Logger, cfg Config, client EthReaderCaller, forwarder Forwarder) *Engine {
	e := &Engine{
		log:       log,
		cfg:       cfg,
		client:    client,
		forwarder: forwarder,
		calls:     make(map[string]Call),
		poller:    newPoller(cfg.PollEvery.Get()),
	}

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
		e.Poll(ctx, time.Now())
	})
}

func (e *Engine) UpdatePreviousEthBlock(height uint64, time uint64) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.prevEthBlock = blockIndex{number: height, time: time}
}

func (e *Engine) initPreviousEthBlock(height uint64, time uint64) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.prevEthBlock == nil {
		e.prevEthBlock = blockIndex{number: height, time: time}
		return true
	}
	return false
}

func (e *Engine) getPreviousEthBlock() blockish {
	e.mu.Lock()
	defer e.mu.Unlock()
	return blockIndex{
		number: e.prevEthBlock.NumberU64(),
		time:   e.prevEthBlock.Time(),
	}
}

func (e *Engine) getCalls() map[string]Call {
	e.mu.Lock()
	defer e.mu.Unlock()
	calls := map[string]Call{}
	for specID, call := range e.calls {
		calls[specID] = call
	}
	return calls
}

func (e *Engine) Stop() {
	// Notify to stop on next iteration.
	e.poller.Stop()
	// Cancel any ongoing queries against Ethereum.
	if e.cancelEthereumQueries != nil {
		e.cancelEthereumQueries()
	}
}

func (e *Engine) GetSpec(id string) (types.EthCallSpec, bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if source, ok := e.calls[id]; ok {
		return source.spec, true
	}

	return types.EthCallSpec{}, false
}

func (e *Engine) MakeResult(specID string, bytes []byte) (Result, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	call, ok := e.calls[specID]
	if !ok {
		return Result{}, fmt.Errorf("no such specification: %v", specID)
	}
	return newResult(call, bytes)
}

func (e *Engine) CallSpec(ctx context.Context, id string, atBlock uint64) (Result, error) {
	e.mu.Lock()
	call, ok := e.calls[id]
	if !ok {
		e.mu.Unlock()
		return Result{}, fmt.Errorf("no such specification: %v", id)
	}
	e.mu.Unlock()

	return call.Call(ctx, e.client, atBlock)
}

func (e *Engine) OnSpecActivated(ctx context.Context, spec types.OracleSpec) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	switch d := spec.ExternalDataSourceSpec.Spec.Data.Content().(type) {
	case types.EthCallSpec:
		id := spec.ExternalDataSourceSpec.Spec.ID
		if _, ok := e.calls[id]; ok {
			return fmt.Errorf("duplicate spec: %s", id)
		}

		ethCall, err := NewCall(d)
		if err != nil {
			return fmt.Errorf("failed to create data source: %w", err)
		}

		e.calls[id] = ethCall
	}

	return nil
}

func (e *Engine) OnSpecDeactivated(ctx context.Context, spec types.OracleSpec) {
	e.mu.Lock()
	defer e.mu.Unlock()
	switch spec.ExternalDataSourceSpec.Spec.Data.Content().(type) {
	case *types.EthCallSpec:
		id := spec.ExternalDataSourceSpec.Spec.ID
		delete(e.calls, id)
	}
}

// Poll is called by the poller in it's own goroutine; it isn't part of the abci code path.
func (e *Engine) Poll(ctx context.Context, wallTime time.Time) {
	// Don't take the mutex here to avoid blocking abci engine while doing potentially lengthy ethereum calls
	// Instead call methods on the engine that take the mutex for a small time where needed.
	// We do need to make use direct use of of e.log, e.client and e.forwarder; but these are static after creation
	// and the methods used are safe for concurrent access.
	lastEthBlock, err := e.client.BlockByNumber(ctx, nil)
	if err != nil {
		e.log.Errorf("failed to get current block header: %w", err)
		return
	}

	e.log.Info("tick",
		logging.Time("wallTime", wallTime),
		logging.BigInt("ethBlock", lastEthBlock.Number()),
		logging.Time("ethTime", time.Unix(int64(lastEthBlock.Time()), 0)))

	// If this is the first time we're running, just set the previous block and return
	if e.initPreviousEthBlock(lastEthBlock.NumberU64(), lastEthBlock.Time()) {
		return
	}

	// Go through an eth blocks one at a time until we get to the most recent one
	for prevEthBlock := e.getPreviousEthBlock(); prevEthBlock.NumberU64() < lastEthBlock.NumberU64(); prevEthBlock = e.getPreviousEthBlock() {
		nextBlockNum := big.NewInt(0).SetUint64(prevEthBlock.NumberU64() + 1)
		nextEthBlock, err := e.client.BlockByNumber(ctx, nextBlockNum)
		if err != nil {
			e.log.Errorf("failed to get next block header: %w", err)
			return
		}

		for specID, call := range e.getCalls() {
			if call.triggered(prevEthBlock, nextEthBlock) {
				res, err := call.Call(ctx, e.client, nextEthBlock.NumberU64())
				if err != nil {
					e.log.Errorf("failed to call contract: %w", err)
					event := makeErrorChainEvent(err.Error(), specID, nextEthBlock)
					e.forwarder.ForwardFromSelf(event)
					continue
				}

				if res.PassesFilters {
					event := makeChainEvent(res, specID, nextEthBlock)
					e.forwarder.ForwardFromSelf(event)
				}
			}
		}
		e.UpdatePreviousEthBlock(nextEthBlock.NumberU64(), nextEthBlock.Time())
	}
}

func makeChainEvent(res Result, specID string, block blockish) *commandspb.ChainEvent {
	ce := commandspb.ChainEvent{
		TxId:  "internal", // NA
		Nonce: 0,          // NA
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

func makeErrorChainEvent(errMsg string, specID string, block blockish) *commandspb.ChainEvent {
	ce := commandspb.ChainEvent{
		TxId:  "internal", // NA
		Nonce: 0,          // NA
		Event: &commandspb.ChainEvent_ContractCall{
			ContractCall: &vega.EthContractCallEvent{
				SpecId:      specID,
				BlockHeight: block.NumberU64(),
				BlockTime:   block.Time(),
				Error:       &errMsg,
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
