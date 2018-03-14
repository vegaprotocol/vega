package blockchain

import (
	"encoding/binary"

	cmn "github.com/tendermint/tmlibs/common"

	"fmt"

	"github.com/tendermint/abci/example/code"
	"github.com/tendermint/abci/server"
	"github.com/tendermint/abci/types"
)

type State struct {
	Size    int64  `json:"size"`
	Height  int64  `json:"height"`
	AppHash []byte `json:"app_hash"`
}

type VegaApplication struct {
	types.BaseApplication

	state State
}

// Starts up a Vega blockchain server.
func Start() error {
	fmt.Println("Starting vega server...")
	vega := NewVegaApplication()
	srv, err := server.NewServer("127.0.0.1:46658", "socket", vega)
	if err != nil {
		return err
	}

	if err := srv.Start(); err != nil {
		return err
	}

	fmt.Println("server started")

	// Wait forever
	cmn.TrapSignal(func() {
		// Cleanup
		srv.Stop()
	})
	return nil

}

func NewVegaApplication() *VegaApplication {
	state := State{}
	return &VegaApplication{state: state}
}

// Stage 1: Mempool Connection
//
// A transaction is received by a validator from a client into
// *one* node's mempool or transaction pool. We need to check whether we consider it
// "legal" (validly formatted, containing non-crazy data from a business
// perspective). If so, send it through.
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
// FIXME: For the moment, just let everything through.
func (app *VegaApplication) CheckTx(tx []byte) types.ResponseCheckTx {
	fmt.Println("Checking transaction (LOCAL): ", string(tx))
	return types.ResponseCheckTx{Code: code.CodeTypeOK}
}

// Stage 2: Consensus Connection
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
// The block header will be updated (TODO) to include some commitment to the
// results of DeliverTx, be it a bitarray of non-OK transactions, or a merkle
// root of the data returned by the DeliverTx requests, or both]
func (app *VegaApplication) DeliverTx(tx []byte) types.ResponseDeliverTx {
	fmt.Println("DeliverTx (ALL NODES): ", string(tx))
	app.state.Size += 1
	return types.ResponseDeliverTx{Code: code.CodeTypeOK}
}

// Commit the block and persist to disk.
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
func (app *VegaApplication) Commit() types.ResponseCommit {
	fmt.Println("committing")
	// Using a memdb - just return the big endian size of the db
	appHash := make([]byte, 8)
	binary.PutVarint(appHash, app.state.Size)
	app.state.AppHash = appHash
	app.state.Height += 1
	fmt.Println("state: ", app.state)
	// saveState(app.state)
	return types.ResponseCommit{Data: appHash}
}
