package tm

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/blockchain/ratelimit"
	"code.vegaprotocol.io/vega/contextutil"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/metrics"
	"code.vegaprotocol.io/vega/proto"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/proto/crypto/keys"
)

const (
	// AbciTxnOK Custom return codes for the abci application, any non-zero code is an error.
	AbciTxnOK uint32 = 0
	// AbciTxnValidationFailure ...
	AbciTxnValidationFailure uint32 = 51
)

type GenesisHandler interface {
	OnGenesis(genesisTime time.Time, appState []byte, validatorsPubkey [][]byte) error
}

// AbciApplication represent the application connection to the chain through the abci api
type AbciApplication struct {
	types.BaseApplication
	Config

	cfgMu     sync.Mutex
	log       *logging.Logger
	processor Processor
	service   ApplicationService
	appHash   []byte
	size      uint64
	txSizes   []int
	txTotals  []uint64

	time            ApplicationTime
	onCriticalError func()

	// metrics
	blockHeightCounter prometheus.Counter

	ghandler  GenesisHandler
	top       ValidatorTopology
	rateLimit *ratelimit.Rates
}

// NewApplication returns a new instance of the Abci application
func NewApplication(log *logging.Logger,
	config Config, proc Processor, svc ApplicationService,
	time ApplicationTime, onCriticalError func(), ghandler GenesisHandler,
	top ValidatorTopology) *AbciApplication {

	// setup logger
	log = log.Named(namedLogger)
	log.SetLevel(config.Level.Get())

	app := AbciApplication{
		log:             log,
		Config:          config,
		processor:       proc,
		service:         svc,
		time:            time,
		onCriticalError: onCriticalError,
		ghandler:        ghandler,
		top:             top,
		rateLimit: ratelimit.New(
			config.RateLimit.Requests,
			config.RateLimit.PerNBlocks,
		),
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

type GenesisState struct {
	Hello string `json:"hello"`
	World uint64 `json:"world"`
}

func (a *AbciApplication) InitChain(req types.RequestInitChain) types.ResponseInitChain {
	vators := make([][]byte, 0, len(req.Validators))
	// get just the pubkeys out of the validator list
	for _, v := range req.Validators {
		var data []byte
		switch t := v.PubKey.Sum.(type) {
		case *keys.PublicKey_Ed25519:
			data = t.Ed25519
		}

		if len(data) > 0 {
			vators = append(vators, data)
		}
	}

	if err := a.ghandler.OnGenesis(req.Time, req.AppStateBytes, vators); err != nil {
		a.log.Error("something happened when initializing vega with the genesis block",
			logging.Error(err))
		// kill the whole application
		a.onCriticalError()
	}

	return types.ResponseInitChain{}
}

// BeginBlock is called by the chain once the new block is starting
func (a *AbciApplication) BeginBlock(beginBlock types.RequestBeginBlock) types.ResponseBeginBlock {
	a.blockHeightCounter.Inc()
	a.rateLimit.NextBlock()

	// We can log more gossiped time info (switchable in config)
	a.cfgMu.Lock()
	if a.LogTimeDebug {
		a.log.Debug("Block time for height",
			logging.Int64("height", beginBlock.Header.Height),
			// TODO: logging.Int64("num-txs", beginBlock.Header.NumTxs),
			logging.Int64("epoch-nano", beginBlock.Header.Time.UnixNano()),
			logging.String("time", beginBlock.Header.Time.String()))
	}
	a.cfgMu.Unlock()

	// Set time provided by ABCI block header (consensus will have been reached on block time)
	epochNow := beginBlock.Header.Time
	// use the hash block as the traceID in the context
	hexBlockHash := hex.EncodeToString(beginBlock.Hash)
	ctx := contextutil.WithTraceID(context.Background(), hexBlockHash)
	a.time.SetTimeNow(ctx, epochNow)

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

// CheckTx is called when a new transaction is being gossiped by the chain to the
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
func (a *AbciApplication) CheckTx(txn types.RequestCheckTx) types.ResponseCheckTx {
	tx, _, err := proto.NewTxFromSignedBundlePayload(txn.Tx)
	if err != nil {
		a.log.Error("Error when decoding payload in CheckTx", logging.Error(err))
		return types.ResponseCheckTx{Code: AbciTxnValidationFailure}
	}

	// Verify ratelimit if node is not a validator
	if !a.top.Exists(tx.GetPubKey()) {
		// Use the Tx's pubkey to verify its rate allowance
		key := ratelimit.Key(tx.GetPubKey()).String()
		if ok := a.rateLimit.Allow(key); !ok {
			a.log.Error("Rate limit exceeded", logging.String("key", key))
			return types.ResponseCheckTx{Code: AbciTxnValidationFailure}
		}
		a.log.Debug("RateLimit allowance", logging.String("key", key), logging.Int("count", a.rateLimit.Count(key)))
	}

	if err := a.processor.Validate(txn.Tx); err != nil {
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
func (a *AbciApplication) DeliverTx(txn types.RequestDeliverTx) types.ResponseDeliverTx {
	a.size++ // Always increment size first, ensure appHash is consistent

	err := a.processor.Process(txn.Tx)
	if err != nil {
		a.log.Error("Error during processing of DeliverTx", logging.Error(err))
		// return types.ResponseDeliverTx{Code: AbciTxnValidationFailure} // todo: revisit this as part of #414
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
	binary.BigEndian.PutUint64(appHash, uint64(a.size))
	a.appHash = appHash

	// Notify the abci/blockchain service imp that the transactions block/batch has completed
	if err := a.service.Commit(); err != nil {
		a.log.Error("Error on blockchain service Commit", logging.Error(err))
	}

	// todo: when an error happens on service commit should we return a different response to ABCI? (#179)
	a.processor.ResetSeenPayloads()
	return types.ResponseCommit{Data: appHash}
}
