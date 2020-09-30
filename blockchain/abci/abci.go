package abci

import (
	"context"
	"encoding/hex"
	"errors"

	"github.com/tendermint/tendermint/abci/types"
)

const (
	// AbciTxnValidationFailure ...
	AbciTxnValidationFailure uint32 = 51

	// AbciTxnDecodingFailure code is returned when CheckTx or DeliverTx fail to decode the Txn.
	AbciTxnDecodingFailure uint32 = 60

	// AbciTxnInternalError code is returned when CheckTx or DeliverTx fail to process the Txn.
	AbciTxnInternalError uint32 = 70

	// AbciUnknownCommandError code is returned when the app doesn't know how to handle a given command.
	AbciUnknownCommandError uint32 = 80
)

func (app *App) InitChain(req types.RequestInitChain) (resp types.ResponseInitChain) {
	if fn := app.OnInitChain; fn != nil {
		return fn(req)
	}
	return
}

func (app *App) BeginBlock(req types.RequestBeginBlock) (resp types.ResponseBeginBlock) {
	app.height = uint64(req.Header.Height)
	if app.replayProtector != nil {
		app.replayProtector.SetHeight(app.height)
	}

	if fn := app.OnBeginBlock; fn != nil {
		app.ctx, resp = fn(req)
	}
	return
}

func (app *App) Commit() (resp types.ResponseCommit) {
	if fn := app.OnCommit; fn != nil {
		return fn()
	}
	return
}

func (app *App) CheckTx(req types.RequestCheckTx) (resp types.ResponseCheckTx) {
	tx, code, err := app.decodeAndValidateTx(req.GetTx())
	if err != nil {
		return NewResponseCheckTx(code, err.Error())
	}

	if err := app.replayProtection(tx); err != nil {
		return NewResponseCheckTx(AbciTxnValidationFailure, err.Error())
	}

	ctx := app.ctx
	if fn := app.OnCheckTx; fn != nil {
		ctx, resp = fn(ctx, req, tx)
		if resp.IsErr() {
			return resp
		}
	}

	// Lookup for check tx, skip if not found
	if fn, ok := app.checkTxs[tx.Command()]; ok {
		if err := fn(ctx, tx); err != nil {
			resp.Code = AbciTxnInternalError
		}
	}

	// at this point we consider the Tx as valid, so we add it to
	// the cache to be consumed by DeliveryTx
	app.cacheTx(&req, tx)
	return resp
}

// replayProtection returns an error when the Tx's BlockHeight distance to the chain is >= than a given threshold.
func (app *App) replayProtection(tx Tx) error {
	// skip replay protection if the Tx didn't specify the block height.
	if tx.BlockHeight() == 0 {
		return nil
	}

	// if this is zero, we assume that the replay protection has not been enabled.
	if app.replayMaxDistance == 0 {
		return nil
	}

	// We perform 2 verifications:

	// First we make sure that the Tx is not on the ring buffer.
	key := hex.EncodeToString(tx.Hash())
	if err := app.replayProtector.Add(key); err != nil {
		return errors.New("tx cached")
	}

	// Then we verify the block distance:

	// If the tx is on a future block, we accept.
	if tx.BlockHeight() > app.height {
		return nil
	}

	// Calculate the distance
	if app.height-tx.BlockHeight() >= uint64(app.replayMaxDistance) {
		return errors.New("tx staled")
	}

	return nil
}

func (app *App) DeliverTx(req types.RequestDeliverTx) (resp types.ResponseDeliverTx) {
	tx := app.txFromCache(&req)
	if tx == nil {
		var (
			code uint32
			err  error
		)
		tx, code, err = app.decodeAndValidateTx(req.GetTx())
		if err != nil {
			return NewResponseDeliverTx(code, err.Error())
		}
	}

	// It's been validated by CheckTx so we can skip the validation here
	ctx := app.ctx
	if fn := app.OnDeliverTx; fn != nil {
		ctx, resp = fn(ctx, req, tx)
		if resp.IsErr() {
			return resp
		}
	}

	// Lookup for deliver tx, fail if not found
	fn := app.deliverTxs[tx.Command()]
	if fn == nil {
		return NewResponseDeliverTx(AbciUnknownCommandError, "")
	}

	if err := fn(ctx, tx); err != nil {
		return NewResponseDeliverTx(AbciTxnInternalError, err.Error())
	}

	return NewResponseDeliverTx(types.CodeTypeOK, "")
}
