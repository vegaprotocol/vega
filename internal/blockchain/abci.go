package blockchain

import (
	"encoding/binary"

	"code.vegaprotocol.io/vega/internal/logging"
	"code.vegaprotocol.io/vega/internal/vegatime"

	"github.com/tendermint/tendermint/abci/types"
)

const (
	// Custom return codes for the abci application, any non-zero code is an error.
	AbciTxnOK                uint32 = 0
	AbciTxnValidationFailure uint32 = 51

	// Maximum sample size for average calculation, used in statistics (average tx per block etc).
	statsSampleSize = 5000
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/application_service_mock.go -package mocks code.vegaprotocol.io/vega/internal/blockchain ApplicationService
type ApplicationService interface {
	Begin() error
	Commit() error
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/application_processor_mock.go -package mocks code.vegaprotocol.io/vega/internal/blockchain ApplicationProcessor
type ApplicationProcessor interface {
	Process(payload []byte) error
	Validate(payload []byte) error
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/application_time_mock.go -package mocks code.vegaprotocol.io/vega/internal/blockchain ApplicationTime
type ApplicationTime interface {
	SetTimeNow(epochTimeNano vegatime.Stamp)
}

type AbciApplication struct {
	types.BaseApplication
	*Config

	stats     *Stats
	processor ApplicationProcessor
	service   ApplicationService
	appHash   []byte
	size      int64
	txSizes   []int
	txTotals  []int

	time            ApplicationTime
	onCriticalError func()
}

func NewApplication(config *Config, stats *Stats, proc ApplicationProcessor, svc ApplicationService, time ApplicationTime, onCriticalError func()) *AbciApplication {
	return &AbciApplication{
		Config:          config,
		stats:           stats,
		processor:       proc,
		service:         svc,
		time:            time,
		onCriticalError: onCriticalError,
	}
}

func (app *AbciApplication) BeginBlock(beginBlock types.RequestBeginBlock) types.ResponseBeginBlock {

	// We can log more gossiped time info (switchable in config)
	if app.LogTimeDebug {
		app.log.Debug("Block time for height",
			logging.Int64("height", beginBlock.Header.Height),
			logging.Int64("num-txs", beginBlock.Header.NumTxs),
			logging.Int64("epoch-nano", beginBlock.Header.Time.UnixNano()),
			logging.String("time", beginBlock.Header.Time.String()))
	}

	// Set time provided by ABCI block header (consensus will have been reached on block time)
	epochNow := beginBlock.Header.Time.UnixNano()
	app.time.SetTimeNow(vegatime.Stamp(epochNow))

	// Notify the abci/blockchain service imp that the transactions block/batch has begun
	if err := app.service.Begin(); err != nil {
		app.log.Error("Failure on blockchain service begin", logging.Error(err))
		app.onCriticalError()
	}

	return types.ResponseBeginBlock{}
}

//func (app *Blockchain) EndBlock(endBlock types.RequestEndBlock) types.ResponseEndBlock {
//	return types.ResponseEndBlock{}
//}

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
func (app *AbciApplication) CheckTx(txn []byte) types.ResponseCheckTx {
	err := app.processor.Validate(txn)
	if err != nil {
		app.log.Error("Error when validating payload in CheckTx", logging.Error(err))
		return types.ResponseCheckTx{Code: AbciTxnValidationFailure}
	}
	return types.ResponseCheckTx{Code: AbciTxnOK}
}

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
func (app *AbciApplication) DeliverTx(txn []byte) types.ResponseDeliverTx {
	err := app.processor.Process(txn)
	if err != nil {
		app.log.Error("Error when validating payload in DeliverTx", logging.Error(err))
		return types.ResponseDeliverTx{Code: AbciTxnValidationFailure}
	}
	txLength := len(txn)
	app.setTxStats(txLength)
	app.size += 1
	return types.ResponseDeliverTx{Code: AbciTxnOK}
}

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
func (app *AbciApplication) Commit() types.ResponseCommit {
	appHash := make([]byte, 8)
	binary.PutVarint(appHash, app.size)
	app.appHash = appHash
	app.stats.height += 1

	// Notify the abci/blockchain service imp that the transactions block/batch has completed
	if err := app.service.Commit(); err != nil {
		app.log.Error("Error on blockchain service Commit", logging.Error(err))
	}

	// todo: when an error happens on service commit should we return a different response to ABCI? (gitlab.com/vega-protocol/trading-core/issues/179)

	app.setBatchStats()
	return types.ResponseCommit{Data: appHash}
}

// setBatchStats is used to calculate any statistics that should be
// recorded once per batch, typically called from commit.
func (app *AbciApplication) setBatchStats() {
	// Calculate the average total txn per batch, over n blocks
	if app.txTotals == nil {
		app.txTotals = make([]int, 0)
	}
	app.txTotals = append(app.txTotals, app.stats.totalTxLastBatch)
	totalTx := 0
	for _, itx := range app.txTotals {
		totalTx += itx
	}
	averageTxTotal := totalTx / len(app.txTotals)

	app.log.Debug("Batch stats for height",
		logging.Uint64("height", app.stats.height),
		logging.Int("average-tx-total", averageTxTotal))

	app.stats.averageTxPerBatch = averageTxTotal
	app.stats.totalTxLastBatch = 0

	// MAX sample size for avg calculation is defined as const.
	if len(app.txTotals) == statsSampleSize {
		app.txTotals = nil
	}
}

// setTxStats is used to calculate any statistics that should be
// recorded once per transaction delivery.
func (app *AbciApplication) setTxStats(txLength int) {
	app.stats.totalTxLastBatch++
	if app.txSizes == nil {
		app.txSizes = make([]int, 0)
	}
	app.txSizes = append(app.txSizes, txLength)
	totalTx := 0
	for _, itx := range app.txSizes {
		totalTx += itx
	}
	averageTxBytes := totalTx / len(app.txSizes)

	app.log.Debug("Transaction stats for height",
		logging.Uint64("height", app.stats.height),
		logging.Int("average-tx-bytes", averageTxBytes))

	app.stats.averageTxSizeBytes = averageTxBytes

	// MAX sample size for avg calculation is defined as const.
	if len(app.txSizes) == statsSampleSize {
		app.txSizes = nil
	}
}
