package abci

import (
	"context"

	"code.vegaprotocol.io/vega/txn"
	"github.com/tendermint/tendermint/abci/types"
)

type Tx interface {
	Command() txn.Command
	Unmarshal(interface{}) error
	PubKey() []byte
	PubKeyHex() string
	Party() string
	Hash() []byte
	Signature() []byte
	Validate() error
	BlockHeight() uint64
	GetCmd() interface{}
}

type Codec interface {
	Decode(in []byte) (Tx, error)
}

// ABCI hooks.
type (
	OnInitChainHandler        func(types.RequestInitChain) types.ResponseInitChain
	OnBeginBlockHandler       func(types.RequestBeginBlock) (context.Context, types.ResponseBeginBlock)
	OnEndBlockHandler         func(types.RequestEndBlock) (context.Context, types.ResponseEndBlock)
	OnCheckTxHandler          func(context.Context, types.RequestCheckTx, Tx) (context.Context, types.ResponseCheckTx)
	OnDeliverTxHandler        func(context.Context, types.RequestDeliverTx, Tx) (context.Context, types.ResponseDeliverTx)
	OnCommitHandler           func() types.ResponseCommit
	ListSnapshotsHandler      func(types.RequestListSnapshots) types.ResponseListSnapshots
	OffserSnapshotHandler     func(types.RequestOfferSnapshot) types.ResponseOfferSnapshot
	LoadSnapshotChunkHandler  func(types.RequestLoadSnapshotChunk) types.ResponseLoadSnapshotChunk
	ApplySnapshotChunkHandler func(context.Context, types.RequestApplySnapshotChunk) types.ResponseApplySnapshotChunk
	InfoHandler               func(types.RequestInfo) types.ResponseInfo
)
