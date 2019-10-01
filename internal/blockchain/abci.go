package blockchain

import (
	"encoding/binary"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/internal/logging"
	"code.vegaprotocol.io/vega/internal/metrics"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/tendermint/tendermint/abci/types"
)

const (
	// AbciTxnOK Custom return codes for the abci application, any non-zero code is an error.
	AbciTxnOK uint32 = 0
	// AbciTxnValidationFailure ...
	AbciTxnValidationFailure uint32 = 51

	// Maximum sample size for average calculation, used in statistics (average tx per block etc).
	statsSampleSize = 5000
)

// ApplicationService ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/application_service_mock.go -package mocks code.vegaprotocol.io/vega/internal/blockchain ApplicationService
type ApplicationService interface {
	Begin() error
	Commit() error
}

// ApplicationProcessor ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/application_processor_mock.go -package mocks code.vegaprotocol.io/vega/internal/blockchain ApplicationProcessor
type ApplicationProcessor interface {
	Process(payload []byte) error
	Validate(payload []byte) error
}

// ApplicationTime ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/application_time_mock.go -package mocks code.vegaprotocol.io/vega/internal/blockchain ApplicationTime
type ApplicationTime interface {
	SetTimeNow(epochTimeNano time.Time)
}

// AbciApplication represent the application connection to the chain through the abci api
type AbciApplication struct {
	types.BaseApplication
	Config

	cfgMu     sync.Mutex
	log       *logging.Logger
	stats     *Stats
	processor ApplicationProcessor
	service   ApplicationService
	appHash   []byte
	size      int64
	txSizes   []int
	txTotals  []int

	time            ApplicationTime
	onCriticalError func()

	// metrics
	blockHeightCounter prometheus.Counter
}

// NewApplication returns a new instance of the Abci application
func NewApplication(log *logging.Logger, config Config, stats *Stats, proc ApplicationProcessor, svc ApplicationService, time ApplicationTime, onCriticalError func()) *AbciApplication {
	// setup logger
	log = log.Named(namedLogger)
	log.SetLevel(config.Level.Get())

	app := AbciApplication{
		log:             log,
		Config:          config,
		stats:           stats,
		processor:       proc,
		service:         svc,
		time:            time,
		onCriticalError: onCriticalError,
	}
	if err := app.setMetrics(); err != nil {
		app.log.Panic(
			"Unable to set up metrics",
			logging.Error(err),
		)
	}

	return &app
}

func (a *AbciApplication) setMetrics() error {
	h, err := metrics.AddInstrument(
		metrics.Counter,
		"block_height_total",
		metrics.Namespace("vega"),
		metrics.Help("Block height"),
	)
	if err != nil {
		return err
	}
	c, err := h.Counter()
	if err != nil {
		return err
	}
	a.blockHeightCounter = c

	return nil
}

// ReloadConf update the internal configuration of the node
func (a *AbciApplication) ReloadConf(cfg Config) {
	a.log.Info("reloading configuration")
	if a.log.GetLevel() != cfg.Level.Get() {
		a.log.Info("updating log level",
			logging.String("old", a.log.GetLevel().String()),
			logging.String("new", cfg.Level.String()),
		)
		a.log.SetLevel(cfg.Level.Get())
	}

	// TODO(): not updating the the actual server for now, may need to look at this later
	// e.g restart the http server on another port or whatever
	a.cfgMu.Lock()
	a.Config = cfg
	a.cfgMu.Unlock()
}

// BeginBlock is called by the chain once the new block is starting
func (a *AbciApplication) BeginBlock(beginBlock types.RequestBeginBlock) types.ResponseBeginBlock {

	a.blockHeightCounter.Inc()
	// We can log more gossiped time info (switchable in config)
	a.cfgMu.Lock()
	if a.LogTimeDebug {
		a.log.Debug("Block time for height",
			logging.Int64("height", beginBlock.Header.Height),
			logging.Int64("num-txs", beginBlock.Header.NumTxs),
			logging.Int64("epoch-nano", beginBlock.Header.Time.UnixNano()),
			logging.String("time", beginBlock.Header.Time.String()))
	}
	a.cfgMu.Unlock()

	// Set time provided by ABCI block header (consensus will have been reached on block time)
	epochNow := beginBlock.Header.Time
	a.time.SetTimeNow(epochNow)

	// Notify the abci/blockchain service imp that the transactions block/batch has begun
	if err := a.service.Begin(); err != nil {
		a.log.Error("Failure on blockchain service begin", logging.Error(err))
		a.onCriticalError()
	}

	return types.ResponseBeginBlock{}
}

//func (app *Blockchain) EndBlock(endBlock types.RequestEndBlock) types.ResponseEndBlock {
//	return types.ResponseEndBlock{}
//}

// CheckTx is called when a new transaction if beeing gossip by the chain to the
// abci application
// Mempool Connection
//
// A transaction is received by a validator from a client into its own
// (*one node*) mempool. We need to check whether we consider it
// "legal" (validly formatted, containing non-crazy data from a business
// perspective). If so, send it through to the consensus round.
//
// From the Tendermint docs:
//
// [The mempool connection is used only for CheckTx requests. Transactions are
// run using CheckTx in the same order they were received by the validator. If
// the CheckTx returns OK, the transaction is kept in memory and relayed to
// other peers in the same order it was received. Otherwise, it is discarded.
//
// CheckTx requests run concurrently with block processing; so they should run
// against a copy of the main application state which is reset after every block.
// This copy is necessary to track transitions made by a sequence of CheckTx
// requests before they are included in a block. When a block is committed,
// the application must ensure to reset the mempool state to the latest
// committed state. Tendermint Core will then filter through all transactions
// in the mempool, removing any that were included in the block, and re-run
// the rest using CheckTx against the post-Commit mempool state]
//
func (a *AbciApplication) CheckTx(txn []byte) types.ResponseCheckTx {
	err := a.processor.Validate(txn)
	if err != nil {
		a.log.Error("Error when validating payload in CheckTx", logging.Error(err))
		return types.ResponseCheckTx{Code: AbciTxnValidationFailure}
	}
	return types.ResponseCheckTx{Code: AbciTxnOK}
}

// DeliverTx is called by the chain once the block have been accepted
// in order to actually deliver the transaction to the abci application
// Consensus Connection
// Step 1: DeliverTx
//
// A transaction has been accepted by more than 2/3 of
// validator nodes. At this step, we can execute our business logic (or,
// in Ethereum terms, this is where the smart contract code lives).
//
// Every honest validator node will run state changes according to what
// happens in this function.
//
// From the Tendermint docs:
//
// [DeliverTx is the workhorse of the blockchain. Tendermint sends the DeliverTx
// requests asynchronously but in order, and relies on the underlying socket
// protocol (ie. TCP) to ensure they are received by the app in order. They
// have already been ordered in the global consensus by the Tendermint protocol.
//
// DeliverTx returns a abci.Result, which includes a Code, Data, and Log. The
// code may be non-zero (non-OK), meaning the corresponding transaction should
// have been rejected by the mempool, but may have been included in a block by
// a Byzantine proposer.
//
// The block header will be updated to include some commitment to the
// results of DeliverTx, be it a bitarray of non-OK transactions, or a merkle
// root of the data returned by the DeliverTx requests, or both]
//
func (a *AbciApplication) DeliverTx(txn []byte) types.ResponseDeliverTx {
	a.size++ // Always increment size first, ensure appHash is consistent
	txLength := len(txn)
	a.setTxStats(txLength)

	err := a.processor.Process(txn)
	if err != nil {
		a.log.Error("Error during processing of DeliverTx", logging.Error(err))
		//return types.ResponseDeliverTx{Code: AbciTxnValidationFailure} // todo: revisit this as part of #414 (gitlab.com/vega-protocol/trading-core/issues/414)
	}

	return types.ResponseDeliverTx{Code: AbciTxnOK}
}

// Commit is called once the block have been accepted, and is persisted in the chain
// Consensus Connection
// Step 2: Commit the block and persist to disk.
//
// From the Tendermint docs:
//
// [Once all processing of the block is complete, Tendermint sends the Commit
// request and blocks waiting for a response. While the mempool may run
// concurrently with block processing (the BeginBlock, DeliverTxs, and
// EndBlock), it is locked for the Commit request so that its state can be
// safely reset during Commit. This means the app MUST NOT do any blocking
// communication with the mempool (ie. broadcast_tx) during Commit, or there
// will be deadlock. Note also that all remaining transactions in the mempool
// are replayed on the mempool connection (CheckTx) following a commit.
//
// The app should respond to the Commit request with a byte array, which is
// the deterministic state root of the application. It is included in the
// header of the next block. It can be used to provide easily verified
// Merkle-proofs of the state of the application.
//
// It is expected that the app will persist state to disk on Commit.
// The option to have all transactions replayed from some previous block is
// the job of the Handshake.
//
func (a *AbciApplication) Commit() types.ResponseCommit {
	appHash := make([]byte, 8)
	binary.PutVarint(appHash, a.size)
	a.appHash = appHash
	a.stats.height++

	// Notify the abci/blockchain service imp that the transactions block/batch has completed
	if err := a.service.Commit(); err != nil {
		a.log.Error("Error on blockchain service Commit", logging.Error(err))
	}

	// todo: when an error happens on service commit should we return a different response to ABCI? (gitlab.com/vega-protocol/trading-core/issues/179)

	a.setBatchStats()
	return types.ResponseCommit{Data: appHash}
}

// setBatchStats is used to calculate any statistics that should be
// recorded once per batch, typically called from commit.
func (a *AbciApplication) setBatchStats() {
	// Calculate the average total txn per batch, over n blocks
	if a.txTotals == nil {
		a.txTotals = make([]int, 0)
	}
	a.txTotals = append(a.txTotals, a.stats.totalTxLastBatch)
	totalTx := 0
	for _, itx := range a.txTotals {
		totalTx += itx
	}
	averageTxTotal := totalTx / len(a.txTotals)

	a.log.Debug("Batch stats for height",
		logging.Uint64("height", a.stats.height),
		logging.Int("average-tx-total", averageTxTotal))

	a.stats.averageTxPerBatch = averageTxTotal
	a.stats.totalTxLastBatch = a.stats.totalTxCurrentBatch
	a.stats.totalTxCurrentBatch = 0

	// MAX sample size for avg calculation is defined as const.
	if len(a.txTotals) == statsSampleSize {
		a.txTotals = nil
	}
}

// setTxStats is used to calculate any statistics that should be
// recorded once per transaction delivery.
func (a *AbciApplication) setTxStats(txLength int) {
	a.stats.totalTxCurrentBatch++
	if a.txSizes == nil {
		a.txSizes = make([]int, 0)
	}
	a.txSizes = append(a.txSizes, txLength)
	totalTx := 0
	for _, itx := range a.txSizes {
		totalTx += itx
	}
	averageTxBytes := totalTx / len(a.txSizes)

	a.log.Debug("Transaction stats for height",
		logging.Uint64("height", a.stats.height),
		logging.Int("average-tx-bytes", averageTxBytes))

	a.stats.averageTxSizeBytes = averageTxBytes

	// MAX sample size for avg calculation is defined as const.
	if len(a.txSizes) == statsSampleSize {
		a.txSizes = nil
	}
}

// Stats - expose unexported stats field, temp fix for testing
func (a *AbciApplication) Stats() *Stats {
	return a.stats
}
