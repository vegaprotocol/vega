package abci

import (
	"context"

	abci "github.com/tendermint/tendermint/abci/types"
)

const (
	// AbciTxnValidationFailure ...
	AbciTxnValidationFailure uint32 = 51

	// AbciTxnDecodingFailure code is returned when CheckTx or DeliverTx fail to decode the Txn.
	AbciTxnDecodingFailure = 60

	// AbciTxnInternalError code is returned when CheckTx or DeliverTx fail to process the Txn.
	AbciTxnInternalError = 70

	// AbciUnknownCommandError code is returned when the app doesn't know how to handle a given command
	AbciUnknownCommandError = 80
)

func (app *App) InitChain(req abci.RequestInitChain) (resp abci.ResponseInitChain) {
	if fn := app.OnInitChain; fn != nil {
		return fn(req)
	}
	return
}

func (app *App) BeginBlock(req abci.RequestBeginBlock) (resp abci.ResponseBeginBlock) {
	if fn := app.OnBeginBlock; fn != nil {
		return fn(req)
	}
	return
}

func (app *App) Commit(req abci.RequestCommit) (resp abci.ResponseCommit) {
	if fn := app.OnCommit; fn != nil {
		return fn(req)
	}
	return
}

func (app *App) CheckTx(req abci.RequestCheckTx) (resp abci.ResponseCheckTx) {
	tx, err := app.codec.Decode(req.GetTx())
	if err != nil {
		return NewResponseCheckTx(AbciTxnDecodingFailure)
	}

	if err := tx.Validate(); err != nil {
		return NewResponseCheckTx(AbciTxnValidationFailure)
	}

	ctx := context.Background()
	if fn := app.OnCheckTx; fn != nil {
		ctx, resp = fn(ctx, req)
		if resp.IsErr() {
			return resp
		}
	}

	// Lookup for check tx, skip if not found
	if fn, ok := app.checkTxs[tx.Command()]; ok {
		if err := fn(ctx, tx); err != nil {
			return NewResponseCheckTx(AbciTxnInternalError)
		}
	}

	return NewResponseCheckTx(abci.CodeTypeOK)
}

func (app *App) DeliverTx(req abci.RequestDeliverTx) (resp abci.ResponseDeliverTx) {
	tx, err := app.codec.Decode(req.GetTx())
	if err != nil {
		return NewResponseDeliverTx(AbciTxnDecodingFailure)
	}

	// It's been validated by CheckTx so we can skip the validation here
	ctx := context.Background()
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

	return NewResponseDeliverTx(abci.CodeTypeOK)
}
