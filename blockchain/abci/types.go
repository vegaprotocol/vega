package abci

import (
	"context"

	"code.vegaprotocol.io/vega/blockchain"
	abci "github.com/tendermint/tendermint/abci/types"
)

type Tx interface {
	Command() blockchain.Command
	Payload() []byte
	PubKey() []byte
	Validate() error
}

type Codec interface {
	Decode(in []byte) (Tx, error)
}

// ABCI hooks
type OnInitChainHandler func(abci.RequestInitChain) abci.ResponseInitChain
type OnBeginBlockHandler func(abci.RequestBeginBlock) abci.ResponseBeginBlock
type OnCheckTxHandler func(context.Context, abci.RequestCheckTx) (context.Context, abci.ResponseCheckTx)
type OnDeliverTxHandler func(context.Context, abci.RequestDeliverTx) (context.Context, abci.ResponseDeliverTx)
type OnCommitHandler func(abci.RequestCommit) abci.ResponseCommit
