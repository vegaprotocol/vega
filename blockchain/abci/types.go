package abci

import (
	"context"

	"code.vegaprotocol.io/vega/blockchain"

	"github.com/tendermint/tendermint/abci/types"
)

type Tx interface {
	Command() blockchain.Command
	Payload() []byte
	PubKey() []byte
	Hash() []byte
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
