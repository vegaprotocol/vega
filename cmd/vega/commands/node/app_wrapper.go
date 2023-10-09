// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package node

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
