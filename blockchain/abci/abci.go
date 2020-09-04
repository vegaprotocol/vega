package abci

import (
	"context"

	"github.com/tendermint/tendermint/abci/types"
)

const (
	// AbciTxnValidationFailure ...
	AbciTxnValidationFailure uint32 = 51
	// AbciTxnDecodingFailure code is returned when CheckTx or DeliverTx fail to decode the Txn.
	AbciTxnDecodingFailure uint32 = 60

	// AbciTxnInternalError code is returned when CheckTx or DeliverTx fail to process the Txn.
	AbciTxnInternalError uint32 = 70

	// AbciUnknownCommandError code is returned when the app doesn't know how to handle a given command
	AbciUnknownCommandError uint32 = 80
)

func (app *App) InitChain(req types.RequestInitChain) (resp types.ResponseInitChain) {
	if fn := app.OnInitChain; fn != nil {
		return fn(req)
	}
	return
}

func (app *App) BeginBlock(req types.RequestBeginBlock) (resp types.ResponseBeginBlock) {
	if fn := app.OnBeginBlock; fn != nil {
		return fn(req)
	}
	return
}

func (app *App) Commit(req types.RequestCommit) (resp types.ResponseCommit) {
	if fn := app.OnCommit; fn != nil {
		return fn(req)
	}
	return
}

func (app *App) CheckTx(req types.RequestCheckTx) (resp types.ResponseCheckTx) {
	tx, err := app.codec.Decode(req.GetTx())
	if err != nil {
		return NewResponseCheckTx(AbciTxnDecodingFailure)
	}

	if err := tx.Validate(); err != nil {
		return NewResponseCheckTx(AbciTxnValidationFailure)
	}

	ctx := TxToContext(context.Background(), tx)
	if fn := app.OnCheckTx; fn != nil {
		ctx, resp = fn(ctx, req)
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

	return resp
}

func (app *App) DeliverTx(req types.RequestDeliverTx) (resp types.ResponseDeliverTx) {
	tx, err := app.codec.Decode(req.GetTx())
	if err != nil {
		return NewResponseDeliverTx(AbciTxnDecodingFailure)
	}

	// It's been validated by CheckTx so we can skip the validation here
	ctx := TxToContext(context.Background(), tx)
	if fn := app.OnDeliverTx; fn != nil {
		ctx, resp = fn(ctx, req)
		if resp.IsErr() {
			return resp
		}
	}

	// Lookup for deliver tx, fail if not found
	fn := app.deliverTxs[tx.Command()]
	if fn == nil {
		return NewResponseDeliverTx(AbciUnknownCommandError)
	}

	if err := fn(ctx, tx); err != nil {
		return NewResponseDeliverTx(AbciTxnInternalError)
	}

	return NewResponseDeliverTx(types.CodeTypeOK)
}
