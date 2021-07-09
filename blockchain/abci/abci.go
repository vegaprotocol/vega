package abci

import (
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
	state, err := LoadGenesisState(req.AppStateBytes)
	if err != nil {
		panic(err)
	}

	if t := state.ReplayAttackThreshold; t != 0 {
		app.replayProtector = NewReplayProtector(t)
	}

	if fn := app.OnInitChain; fn != nil {
		return fn(req)
	}
	return
}

func (app *App) BeginBlock(req types.RequestBeginBlock) (resp types.ResponseBeginBlock) {
	height := uint64(req.Header.Height)
	if app.replayProtector != nil {
		app.replayProtector.SetHeight(height)
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
	tx, code, err := app.getTx(req.GetTx())
	if err != nil {
		return NewResponseCheckTxError(code, err)
	}

	if err := app.replayProtector.CheckTx(tx); err != nil {
		return NewResponseCheckTxError(AbciTxnValidationFailure, err)
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
	if resp.IsOK() {
		app.cacheTx(req.Tx, tx)
	}
	return resp
}

func (app *App) DeliverTx(req types.RequestDeliverTx) (resp types.ResponseDeliverTx) {
	tx, code, err := app.getTx(req.GetTx())
	if err != nil {
		return NewResponseDeliverTxError(code, err)
	}
	app.removeTxFromCache(req.GetTx())

	if err := app.replayProtector.DeliverTx(tx); err != nil {
		return NewResponseDeliverTxError(AbciTxnValidationFailure, err)
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
		return NewResponseDeliverTxError(AbciUnknownCommandError, errors.New("invalid vega command"))
	}

	if err := fn(ctx, tx); err != nil {
		return NewResponseDeliverTxError(AbciTxnInternalError, err)
	}

	return NewResponseDeliverTx(types.CodeTypeOK, "")
}
