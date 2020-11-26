package abci

import (
	"context"

	"code.vegaprotocol.io/vega/txn"
	lru "github.com/hashicorp/golang-lru"
	abci "github.com/tendermint/tendermint/abci/types"
)

type Command byte
type TxHandler func(ctx context.Context, tx Tx) error

type App struct {
	abci.BaseApplication
	codec Codec

	// options
	replayProtector interface {
		SetHeight(uint64)
		DeliverTx(Tx) error
	}

	// handlers
	OnInitChain  OnInitChainHandler
	OnBeginBlock OnBeginBlockHandler
	OnCheckTx    OnCheckTxHandler
	OnDeliverTx  OnDeliverTxHandler
	OnCommit     OnCommitHandler

	// These are Tx handlers
	checkTxs   map[txn.Command]TxHandler
	deliverTxs map[txn.Command]TxHandler

	// checkedTxs holds a map of valid transactions (validated by CheckTx)
	// They are consumed by DeliverTx to avoid double validation.
	checkedTxs *lru.Cache // map[string]Tx

	// the current block context
	ctx context.Context
}

func New(codec Codec) *App {
	lruCache, _ := lru.New(1024)
	return &App{
		codec:           codec,
		replayProtector: &replayProtectorNoop{},
		checkTxs:        map[txn.Command]TxHandler{},
		deliverTxs:      map[txn.Command]TxHandler{},
		checkedTxs:      lruCache,
		ctx:             context.Background(),
	}
}

func (app *App) HandleCheckTx(cmd txn.Command, fn TxHandler) *App {
	app.checkTxs[cmd] = fn
	return app
}

func (app *App) HandleDeliverTx(cmd txn.Command, fn TxHandler) *App {
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
func (app *App) cacheTx(in []byte, tx Tx) {
	app.checkedTxs.Add(string(in), tx)
}

// txFromCache retrieves (and remove if found) a Tx from the cache,
// it returns the Tx or nil if not found.
func (app *App) txFromCache(in []byte) Tx {
	tx, ok := app.checkedTxs.Get(string(in))
	if !ok {
		return nil
	}

	return tx.(Tx)
}

func (app *App) removeTxFromCache(in []byte) {
	app.checkedTxs.Remove(string(in))
}

// getTx returns an internal Tx given a []byte.
// if no errors were found during decoding and validation, the resulting Tx
// will be cached.
// An error code different from 0 is returned if decoding or validation fails
// with its the corresponding error
func (app *App) getTx(bytes []byte) (Tx, uint32, error) {
	if tx := app.txFromCache(bytes); tx != nil {
		return tx, 0, nil
	}

	tx, code, err := app.decodeAndValidateTx(bytes)
	if err != nil {
		return nil, code, err
	}

	return tx, 0, nil
}
