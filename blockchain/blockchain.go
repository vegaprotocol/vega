package blockchain

import (
	"encoding/binary"
	"vega/core"
	"vega/log"
	"vega/msg"
	"github.com/tendermint/tendermint/abci/example/code"
	"github.com/tendermint/tendermint/abci/types"
	"github.com/golang/protobuf/proto"
)

type Blockchain struct {
	types.BaseApplication
	vega *core.Vega
}

func NewBlockchain(vegaApp *core.Vega) *Blockchain {
	return &Blockchain{vega: vegaApp}
}

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
// FIXME: For the moment, just let everything through.
func (app *Blockchain) CheckTx(tx []byte) types.ResponseCheckTx {
	log.Infof("CheckTx: %s", string(tx))
	return types.ResponseCheckTx{Code: code.CodeTypeOK}
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
// The block header will be updated (TODO) to include some commitment to the
// results of DeliverTx, be it a bitarray of non-OK transactions, or a merkle
// root of the data returned by the DeliverTx requests, or both]
func (app *Blockchain) DeliverTx(tx []byte) types.ResponseDeliverTx {
	log.Infof("DeliverTx: %s", string(tx))

	// Decode payload and command
	value, cmd, err := VegaTxDecode(tx)
	if err != nil {
		log.Infof("Invalid tx: %s", string(tx))
		return types.ResponseDeliverTx{Code: code.CodeTypeEncodingError}
	}

	// All incoming messages are order (for now)...
	// deserialize proto msg to struct
	order := msg.OrderPool.Get().(*msg.Order)
	e := proto.Unmarshal(value, order)
	if e != nil {
		log.Infof("Error: Decoding order to proto: ", e.Error())
		return types.ResponseDeliverTx{Code: code.CodeTypeEncodingError}
	}

	// Process known command types
	switch cmd {
		case CreateOrderCommand:
			log.Infof("ABCI received a CREATE ORDER command after consensus")

			// Submit the create new order request to the Vega trading core
			confirmationMessage, errorMessage := app.vega.SubmitOrder(order)
			if confirmationMessage != nil {
				log.Infof("ABCI reports it received an order confirmation message from vega:\n")
				log.Infof("- aggressive order: %+v\n", confirmationMessage.Order)
				log.Infof("- trades: %+v\n", confirmationMessage.Trades)
				log.Infof("- passive orders affected: %+v\n", confirmationMessage.PassiveOrdersAffected)
			}
			if errorMessage != msg.OrderError_NONE {
				log.Infof("ABCI reports it received an order error message from vega:\n")
				log.Infof("- error: %+v\n", errorMessage.String())
			}

		case CancelOrderCommand:
			log.Infof("ABCI received a CANCEL ORDER command after consensus")

			// Submit the create new order request to the Vega trading core
			cancellationMessage, errorMessage := app.vega.CancelOrder(order)
			if cancellationMessage != nil {
				log.Infof("ABCI reports it received an order cancellation message from vega:\n")
				log.Infof("- cancelled order: %+v\n", cancellationMessage.Order)
			}
			if errorMessage != msg.OrderError_NONE {
				log.Infof("ABCI reports it received an order error message from vega:\n")
				log.Infof("- error: %+v\n", errorMessage.String())
			}

		default:
			log.Errorf("UNKNOWN command received after consensus: %v", cmd)
	}

	app.vega.State.Size += 1
	return types.ResponseDeliverTx{Code: code.CodeTypeOK}
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
func (app *Blockchain) Commit() types.ResponseCommit {
	// Using a memdb - just return the big endian size of the db
	appHash := make([]byte, 8)
	binary.PutVarint(appHash, app.vega.State.Size)
	app.vega.State.AppHash = appHash
	app.vega.State.Height += 1

	// saveState(app.state)
	return types.ResponseCommit{Data: appHash}
}


