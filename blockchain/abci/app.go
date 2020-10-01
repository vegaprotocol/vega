package abci

import (
	"context"

	"code.vegaprotocol.io/vega/blockchain"

	"github.com/tendermint/tendermint/abci/types"
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

	// These are Tx handlers
	checkTxs   map[blockchain.Command]TxHandler
	deliverTxs map[blockchain.Command]TxHandler

	// checkedTxs holds a map of valid transactions (validated by CheckTx)
	// They are consumed by DeliverTx to avoid double validation.
	checkedTxs map[string]Tx
}

func New(codec Codec) *App {
	return &App{
		codec:      codec,
		checkTxs:   map[blockchain.Command]TxHandler{},
		deliverTxs: map[blockchain.Command]TxHandler{},
		checkedTxs: map[string]Tx{},
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

// decodeAndValidateTx tries to decode a Tendermint Tx and validate
// it.  It returns the Tx if decoded and validated successfully,
// otherwise it returns the Abci error code and the underlying
// error.
func (app *App) decodeAndValidateTx(bytes []byte) (Tx, uint32, error) {
	tx, err := app.codec.Decode(bytes)
	if err != nil {
		return nil, AbciTxnDecodingFailure, err
	}

	if err := tx.Validate(); err != nil {
		return nil, AbciTxnValidationFailure, err
	}

	return tx, 0, nil
}

// cacheTx adds a Tx to the cache.
func (app *App) cacheTx(r *types.RequestCheckTx, tx Tx) {
	app.checkedTxs[string(r.Tx)] = tx
}

// txFromCache retrieves (and remove if found) a Tx from the cache,
// it returns the Tx or nil if not found.
func (app *App) txFromCache(r *types.RequestDeliverTx) Tx {
	key := string(r.Tx)
	tx, ok := app.checkedTxs[key]
	if !ok {
		return nil
	}
	delete(app.checkedTxs, key)

	return tx
}
