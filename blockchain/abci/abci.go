package abci

import (
	"encoding/hex"
	"errors"

	vgcontext "code.vegaprotocol.io/vega/libs/context"

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

	// AbciSpamError code is returned when CheckTx or DeliverTx fail spam protection tests.
	AbciSpamError uint32 = 89
)

func (app *App) Info(req types.RequestInfo) types.ResponseInfo {
	if fn := app.OnInfo; fn != nil {
		return fn(req)
	}
	return app.BaseApplication.Info(req)
}

func (app *App) InitChain(req types.RequestInitChain) (resp types.ResponseInitChain) {
	_, err := LoadGenesisState(req.AppStateBytes)
	if err != nil {
		panic(err)
	}

	if fn := app.OnInitChain; fn != nil {
		return fn(req)
	}
	return
}

func (app *App) BeginBlock(req types.RequestBeginBlock) (resp types.ResponseBeginBlock) {
	if fn := app.OnBeginBlock; fn != nil {
		app.ctx, resp = fn(req)
	}
	return
}

func (app *App) EndBlock(req types.RequestEndBlock) (resp types.ResponseEndBlock) {
	if fn := app.OnEndBlock; fn != nil {
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
	// first, only decode the transaction but don't validate
	tx, code, err := app.getTx(req.GetTx())
	if err != nil {
		return NewResponseCheckTxError(code, err)
	}

	// check for spam and replay
	if fn := app.OnCheckTxSpam; fn != nil {
		resp = fn(tx)
		if resp.IsErr() {
			return AddCommonCheckTxEvents(resp, tx)
		}
	}

	// if we passed spam protection, validate the signature
	code, err = app.validateTx(tx)
	if err != nil {
		return AddCommonCheckTxEvents(NewResponseCheckTxError(code, err), tx)
	}

	ctx := app.ctx
	if fn := app.OnCheckTx; fn != nil {
		ctx, resp = fn(ctx, req, tx)
		if resp.IsErr() {
			return AddCommonCheckTxEvents(resp, tx)
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

	return AddCommonCheckTxEvents(resp, tx)
}

func (app *App) DeliverTx(req types.RequestDeliverTx) (resp types.ResponseDeliverTx) {
	// first, only decode the transaction but don't validate
	tx, code, err := app.getTx(req.GetTx())
	if err != nil {
		return NewResponseDeliverTxError(code, err)
	}
	app.removeTxFromCache(req.GetTx())

	// check for spam and replay
	if fn := app.OnDeliverTxSpam; fn != nil {
		resp = fn(tx)
		if resp.IsErr() {
			return AddCommonDeliverTxEvents(resp, tx)
		}
	}

	// if we passed spam protection, validate the signature
	code, err = app.validateTx(tx)
	if err != nil {
		return NewResponseDeliverTxError(code, err)
	}

	// It's been validated by CheckTx so we can skip the validation here
	ctx := app.ctx
	if fn := app.OnDeliverTx; fn != nil {
		ctx, resp = fn(ctx, req, tx)
		if resp.IsErr() {
			return AddCommonDeliverTxEvents(resp, tx)
		}
	}

	// Lookup for deliver tx, fail if not found
	fn := app.deliverTxs[tx.Command()]
	if fn == nil {
		return AddCommonDeliverTxEvents(
			NewResponseDeliverTxError(AbciUnknownCommandError, errors.New("invalid vega command")), tx,
		)
	}

	txHash := hex.EncodeToString(tx.Hash())
	ctx = vgcontext.WithTxHash(ctx, txHash)

	if err := fn(ctx, tx); err != nil {
		return AddCommonDeliverTxEvents(
			NewResponseDeliverTxError(AbciTxnInternalError, err), tx,
		)
	}

	return AddCommonDeliverTxEvents(
		NewResponseDeliverTx(types.CodeTypeOK, ""), tx,
	)
}

func (app *App) ListSnapshots(req types.RequestListSnapshots) (resp types.ResponseListSnapshots) {
	if app.OnListSnapshots != nil {
		resp = app.OnListSnapshots(req)
	}
	return
}

func (app *App) OfferSnapshot(req types.RequestOfferSnapshot) (resp types.ResponseOfferSnapshot) {
	if app.OnOfferSnapshot != nil {
		resp = app.OnOfferSnapshot(req)
	}
	return
}

func (app *App) LoadSnapshotChunk(req types.RequestLoadSnapshotChunk) (resp types.ResponseLoadSnapshotChunk) {
	if app.OnLoadSnapshotChunk != nil {
		resp = app.OnLoadSnapshotChunk(req)
	}
	return
}

func (app *App) ApplySnapshotChunk(req types.RequestApplySnapshotChunk) (resp types.ResponseApplySnapshotChunk) {
	if app.OnApplySnapshotChunk != nil {
		resp = app.OnApplySnapshotChunk(app.ctx, req)
	}
	return
}

func AddCommonCheckTxEvents(resp types.ResponseCheckTx, tx Tx) types.ResponseCheckTx {
	resp.Events = getBaseTxEvents(tx)
	return resp
}

func AddCommonDeliverTxEvents(resp types.ResponseDeliverTx, tx Tx) types.ResponseDeliverTx {
	resp.Events = getBaseTxEvents(tx)
	return resp
}

func getBaseTxEvents(tx Tx) []types.Event {
	return []types.Event{
		{
			Type: "tx",
			Attributes: []types.EventAttribute{
				{
					Key:   []byte("submitter"),
					Value: []byte(tx.PubKeyHex()),
					Index: true,
				},
			},
		},
		{
			Type: "command",
			Attributes: []types.EventAttribute{
				{
					Key:   []byte("type"),
					Value: []byte(tx.Command().String()),
					Index: true,
				},
			},
		},
	}
}
