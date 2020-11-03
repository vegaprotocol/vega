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
	Hash() []byte
	Signature() []byte
	Validate() error
	BlockHeight() uint64
}

type Codec interface {
	Decode(in []byte) (Tx, error)
}

// ABCI hooks
type OnInitChainHandler func(types.RequestInitChain) types.ResponseInitChain
type OnBeginBlockHandler func(types.RequestBeginBlock) (context.Context, types.ResponseBeginBlock)
type OnCheckTxHandler func(context.Context, types.RequestCheckTx, Tx) (context.Context, types.ResponseCheckTx)
type OnDeliverTxHandler func(context.Context, types.RequestDeliverTx, Tx) (context.Context, types.ResponseDeliverTx)
type OnCommitHandler func() types.ResponseCommit
