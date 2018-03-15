package blockchain

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"

	"fmt"

	"github.com/tendermint/abci/example/code"
	"github.com/tendermint/abci/types"

	"vega/core"
	"vega/proto"
)

type State struct {
	Size    int64  `json:"size"`
	Height  int64  `json:"height"`
	AppHash []byte `json:"app_hash"`
}

type Blockchain struct {
	types.BaseApplication

	vega  core.Vega
	state State
}

func NewBlockchain(vegaApp core.Vega) *Blockchain {
	state := State{}
	return &Blockchain{state: state, vega: vegaApp}
}

// Stage 1: Mempool Connection
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
// FIXME: For the moment, just let everything through.
func (app *Blockchain) CheckTx(tx []byte) types.ResponseCheckTx {
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
func (app *Blockchain) DeliverTx(tx []byte) types.ResponseDeliverTx {
	fmt.Println("DeliverTx (ALL NODES): ", string(tx))

	// split the transaction
	var key, value []byte
	parts := bytes.Split(tx, []byte("="))
	if len(parts) == 2 {
		key, value = parts[0], parts[1]
	} else {
		return types.ResponseDeliverTx{Code: code.CodeTypeEncodingError}
	}
	fmt.Println("Got key: ", string(key))
	fmt.Println("About to try and decode: ", string(value))
	// decode base64
	var jsonBlob, err = base64.URLEncoding.DecodeString(string(value))
	if err != nil {
		fmt.Println("Error decoding: " + err.Error())
	}
	fmt.Println("Decoded: ", string(jsonBlob))
	// deserialize JSON to struct
	var order msg.Order
	e := json.Unmarshal(jsonBlob, &order)
	if e != nil {
		fmt.Println("Error: ", e.Error())
	}

	res, _ := app.vega.SubmitOrder(order)
	fmt.Println("DeliverTx response: ", res)

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
func (app *Blockchain) Commit() types.ResponseCommit {
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
