package abci

import (
	"context"

	"code.vegaprotocol.io/vega/blockchain"

	abci "github.com/tendermint/tendermint/abci/types"
)

type Command byte
type TxHandler func(ctx context.Context, tx Tx) error

type App struct {
	abci.BaseApplication
	codec Codec

	OnInitChain  OnInitChainHandler
	OnBeginBlock OnBeginBlockHandler
	OnCheckTx    OnCheckTxHandler
	OnDeliverTx  OnDeliverTxHandler
	OnCommit     OnCommitHandler

	checkTxs   map[blockchain.Command]TxHandler
	deliverTxs map[blockchain.Command]TxHandler
}

func New(codec Codec) *App {
	return &App{
		codec:      codec,
		checkTxs:   map[blockchain.Command]TxHandler{},
		deliverTxs: map[blockchain.Command]TxHandler{},
	}
}

func (app *App) HandleCheckTx(cmd blockchain.Command, fn TxHandler) *App {
	app.checkTxs[cmd] = fn
	return app
}

func (app *App) HandleDeliverTx(cmd blockchain.Command, fn TxHandler) *App {
	app.deliverTxs[cmd] = fn
	return app
}
