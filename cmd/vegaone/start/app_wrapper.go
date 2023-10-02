// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package start

import "github.com/tendermint/tendermint/abci/types"

type appW struct {
	// this is the application currently in use
	impl types.Application
	// this is the application that'll need to be swap if expected by
	// the application, will be call on commit and needs to be swap atomically
	// before returning from Commit
	update types.Application
}

func newAppW(app types.Application) *appW {
	return &appW{
		impl: app,
	}
}

func (app *appW) Info(req types.RequestInfo) types.ResponseInfo {
	return app.impl.Info(req)
}

func (app *appW) DeliverTx(req types.RequestDeliverTx) types.ResponseDeliverTx {
	return app.impl.DeliverTx(req)
}

func (app *appW) CheckTx(req types.RequestCheckTx) types.ResponseCheckTx {
	return app.impl.CheckTx(req)
}

func (app *appW) Commit() types.ResponseCommit {
	resp := app.impl.Commit()
	// if we are scheduled for an upgrade of the protocol
	// let's do it now.
	if app.update != nil {
		app.impl = app.update
		app.update = nil
	}
	return resp
}

func (app *appW) Query(req types.RequestQuery) types.ResponseQuery {
	return app.impl.Query(req)
}

func (app *appW) InitChain(req types.RequestInitChain) types.ResponseInitChain {
	return app.impl.InitChain(req)
}

func (app *appW) BeginBlock(req types.RequestBeginBlock) types.ResponseBeginBlock {
	return app.impl.BeginBlock(req)
}

func (app *appW) EndBlock(req types.RequestEndBlock) types.ResponseEndBlock {
	return app.impl.EndBlock(req)
}

func (app *appW) ListSnapshots(
	req types.RequestListSnapshots,
) types.ResponseListSnapshots {
	return app.impl.ListSnapshots(req)
}

func (app *appW) OfferSnapshot(
	req types.RequestOfferSnapshot,
) types.ResponseOfferSnapshot {
	return app.impl.OfferSnapshot(req)
}

func (app *appW) LoadSnapshotChunk(
	req types.RequestLoadSnapshotChunk,
) types.ResponseLoadSnapshotChunk {
	return app.impl.LoadSnapshotChunk(req)
}

func (app *appW) ApplySnapshotChunk(
	req types.RequestApplySnapshotChunk,
) types.ResponseApplySnapshotChunk {
	return app.impl.ApplySnapshotChunk(req)
}

func (app *appW) SetOption(
	req types.RequestSetOption,
) types.ResponseSetOption {
	return app.impl.SetOption(req)
}
