package abci

import (
	"context"

	"code.vegaprotocol.io/vega/blockchain"
	abci "github.com/tendermint/tendermint/abci/types"
)

type Command byte
type CheckTxHandler func(ctx context.Context, tx Tx) error
type DeliverTxHandler func(ctx context.Context, tx Tx) error

type App struct {
	abci.BaseApplication
	codec Codec

	OnInitChain  OnInitChainHandler
	OnBeginBlock OnBeginBlockHandler
	OnCheckTx    OnCheckTxHandler
	OnDeliverTx  OnDeliverTxHandler
	OnCommit     OnCommitHandler

	checkTxs   map[blockchain.Command]CheckTxHandler
	deliverTxs map[blockchain.Command]DeliverTxHandler
}

func New(codec Codec) *App {
	return &App{
		codec:      codec,
		checkTxs:   map[blockchain.Command]CheckTxHandler{},
		deliverTxs: map[blockchain.Command]DeliverTxHandler{},
	}
}

func (app *App) HandleCheckTx(cmd blockchain.Command, fn CheckTxHandler) *App {
	app.checkTxs[cmd] = fn
	return app
}

func (app *App) HandleDeliverTx(cmd blockchain.Command, fn DeliverTxHandler) *App {
	app.deliverTxs[cmd] = fn
	return app
}
