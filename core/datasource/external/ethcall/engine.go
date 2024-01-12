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

package ethcall

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"reflect"
	"strconv"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/core/datasource"
	"code.vegaprotocol.io/vega/core/datasource/external/ethcall/common"
	"code.vegaprotocol.io/vega/libs/ptr"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/protos/vega"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"

	"github.com/ethereum/go-ethereum"
)

type EthReaderCaller interface {
	ethereum.ContractCaller
	ethereum.ChainReader
	ChainID(context.Context) (*big.Int, error)
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/forwarder_mock.go -package mocks code.vegaprotocol.io/vega/core/datasource/external/ethcall Forwarder
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
	isValidator           bool
	client                EthReaderCaller
	calls                 map[string]Call
	forwarder             Forwarder
	prevEthBlock          blockish
	cancelEthereumQueries context.CancelFunc
	poller                *poller
	mu                    sync.Mutex

	chainID uint64
}

func NewEngine(log *logging.Logger, cfg Config, isValidator bool, client EthReaderCaller, forwarder Forwarder) *Engine {
	e := &Engine{
		log:         log,
		cfg:         cfg,
		isValidator: isValidator,
		client:      client,
		forwarder:   forwarder,
		calls:       make(map[string]Call),
		poller:      newPoller(cfg.PollEvery.Get()),
	}
	return e
}

// EnsureChainID tells the engine which chainID it should be related to, and it confirms this against the its client.
func (e *Engine) EnsureChainID(chainID string) {
	e.chainID, _ = strconv.ParseUint(chainID, 10, 64)

	// if the node is a validator, we now check the chainID against the chain the client is connected to.
	if e.isValidator {
		cid, err := e.client.ChainID(context.Background())
		if err != nil {
			log.Panic("could not load chain ID", logging.Error(err))
		}

		if cid.Uint64() != e.chainID {
			log.Panic("chain ID mismatch between ethCall engine and EVM client",
				logging.Uint64("client-chain-id", cid.Uint64()),
				logging.Uint64("engine-chain-id", e.chainID),
			)
		}
	}
}

// Start starts the polling of the Ethereum bridges, listens to the events
// they emit and forward it to the network.
func (e *Engine) Start() {
	if e.isValidator && !reflect.ValueOf(e.client).IsNil() {
		go func() {
			ctx, cancelEthereumQueries := context.WithCancel(context.Background())
			defer cancelEthereumQueries()

			e.cancelEthereumQueries = cancelEthereumQueries

			if e.log.IsDebug() {
				e.log.Debug("Starting ethereum contract call polling engine")
			}

			e.poller.Loop(func() {
				e.Poll(ctx, time.Now())
			})
		}()
	}
}

func (e *Engine) StartAtHeight(height uint64, time uint64) {
	e.prevEthBlock = blockIndex{number: height, time: time}
	e.Start()
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

func (e *Engine) GetSpec(id string) (common.Spec, bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if source, ok := e.calls[id]; ok {
		return source.spec, true
	}

	return common.Spec{}, false
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

func (e *Engine) GetEthTime(ctx context.Context, atBlock uint64) (uint64, error) {
	blockNum := big.NewInt(0).SetUint64(atBlock)
	header, err := e.client.HeaderByNumber(ctx, blockNum)
	if err != nil {
		return 0, fmt.Errorf("failed to get block header: %w", err)
	}

	if header == nil {
		return 0, fmt.Errorf("nil block header: %w", err)
	}

	return header.Time, nil
}

func (e *Engine) GetRequiredConfirmations(id string) (uint64, error) {
	e.mu.Lock()
	call, ok := e.calls[id]
	if !ok {
		e.mu.Unlock()
		return 0, fmt.Errorf("no such specification: %v", id)
	}
	e.mu.Unlock()

	return call.spec.RequiredConfirmations, nil
}

func (e *Engine) GetInitialTriggerTime(id string) (uint64, error) {
	e.mu.Lock()
	call, ok := e.calls[id]
	if !ok {
		e.mu.Unlock()
		return 0, fmt.Errorf("no such specification: %v", id)
	}
	e.mu.Unlock()

	return call.initialTime(), nil
}

func (e *Engine) OnSpecActivated(ctx context.Context, spec datasource.Spec) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	switch d := spec.Data.Content().(type) {
	case common.Spec:
		id := spec.ID
		if _, ok := e.calls[id]; ok {
			return fmt.Errorf("duplicate spec: %s", id)
		}

		ethCall, err := NewCall(d)
		if err != nil {
			return fmt.Errorf("failed to create data source: %w", err)
		}

		// here ensure we are on the engine with the right network ID
		// not an error, just return
		if e.chainID != d.L2ChainID {
			return nil
		}

		e.calls[id] = ethCall
	}

	return nil
}

func (e *Engine) OnSpecDeactivated(ctx context.Context, spec datasource.Spec) {
	e.mu.Lock()
	defer e.mu.Unlock()
	switch spec.Data.Content().(type) {
	case common.Spec:
		id := spec.ID
		delete(e.calls, id)
	}
}

// Poll is called by the poller in it's own goroutine; it isn't part of the abci code path.
func (e *Engine) Poll(ctx context.Context, wallTime time.Time) {
	// Don't take the mutex here to avoid blocking abci engine while doing potentially lengthy ethereum calls
	// Instead call methods on the engine that take the mutex for a small time where needed.
	// We do need to make use direct use of of e.log, e.client and e.forwarder; but these are static after creation
	// and the methods used are safe for concurrent access.
	lastEthBlock, err := e.client.HeaderByNumber(ctx, nil)
	if err != nil {
		e.log.Error("failed to get current block header", logging.Error(err))
		return
	}

	e.log.Info("tick",
		logging.Uint64("chainID", e.chainID),
		logging.Time("wallTime", wallTime),
		logging.BigInt("ethBlock", lastEthBlock.Number),
		logging.Time("ethTime", time.Unix(int64(lastEthBlock.Time), 0)))

	// If the previous eth block has not been set, set it to the current eth block
	if e.prevEthBlock == nil {
		e.prevEthBlock = blockIndex{number: lastEthBlock.Number.Uint64(), time: lastEthBlock.Time}
	}

	// Go through an eth blocks one at a time until we get to the most recent one
	for prevEthBlock := e.prevEthBlock; prevEthBlock.NumberU64() < lastEthBlock.Number.Uint64(); prevEthBlock = e.prevEthBlock {
		nextBlockNum := big.NewInt(0).SetUint64(prevEthBlock.NumberU64() + 1)
		nextEthBlock, err := e.client.HeaderByNumber(ctx, nextBlockNum)
		if err != nil {
			e.log.Error("failed to get next block header", logging.Error(err))
			return
		}

		nextEthBlockIsh := blockIndex{number: nextEthBlock.Number.Uint64(), time: nextEthBlock.Time}
		for specID, call := range e.getCalls() {
			if call.triggered(prevEthBlock, nextEthBlockIsh) {
				res, err := call.Call(ctx, e.client, nextEthBlock.Number.Uint64())
				if err != nil {
					e.log.Error("failed to call contract", logging.Error(err))
					event := makeErrorChainEvent(err.Error(), specID, nextEthBlockIsh, e.chainID)
					e.forwarder.ForwardFromSelf(event)
					continue
				}

				if res.PassesFilters {
					event := makeChainEvent(res, specID, nextEthBlockIsh, e.chainID)
					e.forwarder.ForwardFromSelf(event)
				}
			}
		}

		e.prevEthBlock = nextEthBlockIsh
	}
}

func makeChainEvent(res Result, specID string, block blockish, chainID uint64) *commandspb.ChainEvent {
	ce := commandspb.ChainEvent{
		TxId:  "internal", // NA
		Nonce: 0,          // NA
		Event: &commandspb.ChainEvent_ContractCall{
			ContractCall: &vega.EthContractCallEvent{
				SpecId:      specID,
				BlockHeight: block.NumberU64(),
				BlockTime:   block.Time(),
				Result:      res.Bytes,
				L2ChainId:   ptr.From(chainID),
			},
		},
	}

	return &ce
}

func makeErrorChainEvent(errMsg string, specID string, block blockish, chainID uint64) *commandspb.ChainEvent {
	ce := commandspb.ChainEvent{
		TxId:  "internal", // NA
		Nonce: 0,          // NA
		Event: &commandspb.ChainEvent_ContractCall{
			ContractCall: &vega.EthContractCallEvent{
				SpecId:      specID,
				BlockHeight: block.NumberU64(),
				BlockTime:   block.Time(),
				Error:       &errMsg,
				L2ChainId:   ptr.From(chainID),
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
	done      chan bool
	pollEvery time.Duration
}

func newPoller(pollEvery time.Duration) *poller {
	return &poller{
		done:      make(chan bool, 1),
		pollEvery: pollEvery,
	}
}

// Loop starts the poller loop until it's broken, using the Stop method.
func (s *poller) Loop(fn func()) {
	ticker := time.NewTicker(s.pollEvery)
	defer func() {
		ticker.Stop()
		ticker.Reset(s.pollEvery)
	}()

	for {
		select {
		case <-s.done:
			return
		case <-ticker.C:
			fn()
		}
	}
}

// Stop stops the poller loop.
func (s *poller) Stop() {
	s.done <- true
}
