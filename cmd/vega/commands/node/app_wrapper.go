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

import (
	"context"

	"github.com/cometbft/cometbft/abci/types"
)

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

func (app *appW) Info(ctx context.Context, req *types.RequestInfo) (*types.ResponseInfo, error) {
	return app.impl.Info(ctx, req)
}

func (app *appW) CheckTx(ctx context.Context, req *types.RequestCheckTx) (*types.ResponseCheckTx, error) {
	return app.impl.CheckTx(ctx, req)
}

func (app *appW) Commit(ctx context.Context, req *types.RequestCommit) (*types.ResponseCommit, error) {
	resp, err := app.impl.Commit(ctx, req)
	// if we are scheduled for an upgrade of the protocol
	// let's do it now.
	if app.update != nil {
		app.impl = app.update
		app.update = nil
	}
	return resp, err
}

func (app *appW) Query(ctx context.Context, req *types.RequestQuery) (*types.ResponseQuery, error) {
	return app.impl.Query(ctx, req)
}

func (app *appW) InitChain(ctx context.Context, req *types.RequestInitChain) (*types.ResponseInitChain, error) {
	return app.impl.InitChain(ctx, req)
}

func (app *appW) ListSnapshots(ctx context.Context, req *types.RequestListSnapshots) (*types.ResponseListSnapshots, error) {
	return app.impl.ListSnapshots(ctx, req)
}

func (app *appW) OfferSnapshot(ctx context.Context, req *types.RequestOfferSnapshot) (*types.ResponseOfferSnapshot, error) {
	return app.impl.OfferSnapshot(ctx, req)
}

func (app *appW) LoadSnapshotChunk(ctx context.Context, req *types.RequestLoadSnapshotChunk) (*types.ResponseLoadSnapshotChunk, error) {
	return app.impl.LoadSnapshotChunk(ctx, req)
}

func (app *appW) ApplySnapshotChunk(ctx context.Context, req *types.RequestApplySnapshotChunk) (*types.ResponseApplySnapshotChunk, error) {
	return app.impl.ApplySnapshotChunk(ctx, req)
}

func (app *appW) PrepareProposal(ctx context.Context, proposal *types.RequestPrepareProposal) (*types.ResponsePrepareProposal, error) {
	return app.impl.PrepareProposal(ctx, proposal)
}

func (app *appW) ProcessProposal(ctx context.Context, proposal *types.RequestProcessProposal) (*types.ResponseProcessProposal, error) {
	return app.impl.ProcessProposal(ctx, proposal)
}

func (app *appW) FinalizeBlock(ctx context.Context, req *types.RequestFinalizeBlock) (*types.ResponseFinalizeBlock, error) {
	return app.impl.FinalizeBlock(ctx, req)
}

func (app *appW) ExtendVote(ctx context.Context, req *types.RequestExtendVote) (*types.ResponseExtendVote, error) {
	return app.impl.ExtendVote(ctx, req)
}

func (app *appW) VerifyVoteExtension(ctx context.Context, req *types.RequestVerifyVoteExtension) (*types.ResponseVerifyVoteExtension, error) {
	return app.impl.VerifyVoteExtension(ctx, req)
}
