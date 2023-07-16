// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package abci

import (
	"context"
	"encoding/hex"
	"strconv"

	"code.vegaprotocol.io/vega/core/blockchain"
	vgcontext "code.vegaprotocol.io/vega/libs/context"

	"github.com/cometbft/cometbft/abci/types"
)

func (app *App) Info(ctx context.Context, req *types.RequestInfo) (*types.ResponseInfo, error) {
	if fn := app.OnInfo; fn != nil {
		return fn(ctx, req)
	}
	return app.BaseApplication.Info(ctx, req)
}

func (app *App) InitChain(_ context.Context, req *types.RequestInitChain) (*types.ResponseInitChain, error) {
	_, err := LoadGenesisState(req.AppStateBytes)
	if err != nil {
		panic(err)
	}

	if fn := app.OnInitChain; fn != nil {
		return fn(req)
	}
	return &types.ResponseInitChain{}, nil
}

// PrepareProposal will take the given transactions from the mempool and attempts to prepare a
// proposal from them when it's our turn to do so while keeping the size, gas, pow, and spam constraints.
func (app *App) PrepareProposal(_ context.Context, req *types.RequestPrepareProposal) (*types.ResponsePrepareProposal, error) {
	txs := make([]Tx, 0, len(req.Txs))
	rawTxs := make([][]byte, 0, len(req.Txs))
	for _, v := range req.Txs {
		tx, _, err := app.getTx(v)
		// ignore transactions we can't verify
		if err != nil {
			continue
		}
		// ignore transactions we don't know to handle
		if _, ok := app.deliverTxs[tx.Command()]; !ok {
			continue
		}
		txs = append(txs, tx)
		rawTxs = append(rawTxs, v)
	}

	// let the application decide on the order and the number of transactions it wants to pick up for this block
	res := &types.ResponsePrepareProposal{Txs: app.OnPrepareProposal(txs, rawTxs)}
	return res, nil
}

// ProcessProposal implements part of the Application interface.
// It accepts any proposal that does not contain a malformed transaction.
// NB: processProposal will not be called if the node is fast-sync-ing so no state change is allowed here!!!.
func (app *App) ProcessProposal(_ context.Context, req *types.RequestProcessProposal) (*types.ResponseProcessProposal, error) {
	// check transaction signatures if any is wrong, reject the block
	txs := make([]Tx, 0, len(req.Txs))
	for _, v := range req.Txs {
		tx, _, err := app.getTx(v)
		if err != nil {
			// if there's a transaction we can't decode or verify, reject it
			return &types.ResponseProcessProposal{Status: types.ResponseProcessProposal_REJECT}, err
		}
		// if there's no handler for a transaction, reject it
		if _, ok := app.deliverTxs[tx.Command()]; !ok {
			return &types.ResponseProcessProposal{Status: types.ResponseProcessProposal_REJECT}, nil
		}
		txs = append(txs, tx)
	}
	// let the application verify the block
	if !app.OnProcessProposal(txs) {
		return &types.ResponseProcessProposal{Status: types.ResponseProcessProposal_REJECT}, nil
	}
	return &types.ResponseProcessProposal{Status: types.ResponseProcessProposal_ACCEPT}, nil
}

func (app *App) Commit(_ context.Context, req *types.RequestCommit) (*types.ResponseCommit, error) {
	if fn := app.OnCommit; fn != nil {
		return fn()
	}
	return &types.ResponseCommit{}, nil
}

func (app *App) CheckTx(_ context.Context, req *types.RequestCheckTx) (*types.ResponseCheckTx, error) {
	// first, only decode the transaction but don't validate
	tx, code, err := app.getTx(req.GetTx())
	var resp *types.ResponseCheckTx
	if err != nil {
		// TODO I think we need to return error in this case as now the API allows for it
		// return blockchain.NewResponseCheckTxError(code, err), err
		return blockchain.NewResponseCheckTxError(code, err), nil
	}

	// check for spam and replay
	if fn := app.OnCheckTxSpam; fn != nil {
		resp := fn(tx)
		if resp.IsErr() {
			return AddCommonCheckTxEvents(&resp, tx), nil
		}
	}

	ctx := app.ctx
	if fn := app.OnCheckTx; fn != nil {
		ctx, resp = fn(ctx, req, tx)
		if resp.IsErr() {
			return AddCommonCheckTxEvents(resp, tx), nil
		}
	}

	// Lookup for check tx, skip if not found
	if fn, ok := app.checkTxs[tx.Command()]; ok {
		if err := fn(ctx, tx); err != nil {
			return AddCommonCheckTxEvents(blockchain.NewResponseCheckTxError(blockchain.AbciTxnInternalError, err), tx), nil
		}
	}

	// at this point we consider the Transaction as valid, so we add it to
	// the cache to be consumed by DeliveryTx
	if resp.IsOK() {
		app.cacheTx(req.Tx, tx)
	}
	return AddCommonCheckTxEvents(resp, tx), nil
}

// FinalizeBlock lets the application process a whole block end to end.
func (app *App) FinalizeBlock(_ context.Context, req *types.RequestFinalizeBlock) (*types.ResponseFinalizeBlock, error) {
	blockHeight := uint64(req.Height)
	blockTime := req.Time

	txs := make([]Tx, 0, len(req.Txs))
	for _, rtx := range req.Txs {
		// getTx can't fail at this point as we've verified on processProposal
		tx, _, _ := app.getTx(rtx)
		app.removeTxFromCache(rtx)
		txs = append(txs, tx)
	}

	app.ctx = app.OnBeginBlock(blockHeight, hex.EncodeToString(req.Hash), blockTime, hex.EncodeToString(req.ProposerAddress), txs)
	results := make([]*types.ExecTxResult, 0, len(req.Txs))
	events := []types.Event{}

	for _, tx := range txs {
		// there must be a handling function at this point
		fn := app.deliverTxs[tx.Command()]
		txHash := hex.EncodeToString(tx.Hash())
		ctx := vgcontext.WithTxHash(app.ctx, txHash)
		// process the transaction and handle errors
		if err := fn(ctx, tx); err != nil {
			if perr, ok := err.(MaybePartialError); ok && perr.IsPartial() {
				results = append(results, blockchain.NewResponseDeliverTxError(blockchain.AbciTxnPartialProcessingError, err))
			} else {
				results = append(results, blockchain.NewResponseDeliverTxError(blockchain.AbciTxnInternalError, err))
			}
		} else {
			results = append(results, blockchain.NewResponseDeliverTx(types.CodeTypeOK, ""))
		}
		events = append(events, getBaseTxEvents(tx)...)
	}
	valUpdates, consensusUpdates := app.OnEndBlock(blockHeight)
	events = append(events, types.Event{
		Type: "val_updates",
		Attributes: []types.EventAttribute{
			{
				Key:   "size",
				Value: strconv.Itoa(valUpdates.Len()),
			},
			{
				Key:   "height",
				Value: strconv.Itoa(int(req.Height)),
			},
		},
	},
	)

	hash := app.OnFinalize()
	println("finished processing block", blockHeight, "with block hash", hex.EncodeToString(hash))
	return &types.ResponseFinalizeBlock{
		TxResults:             results,
		ValidatorUpdates:      valUpdates,
		ConsensusParamUpdates: &consensusUpdates,
		AppHash:               hash,
		Events:                events,
	}, nil
}

func (app *App) ListSnapshots(ctx context.Context, req *types.RequestListSnapshots) (*types.ResponseListSnapshots, error) {
	if app.OnListSnapshots != nil {
		return app.OnListSnapshots(ctx, req)
	}
	return &types.ResponseListSnapshots{}, nil
}

func (app *App) OfferSnapshot(ctx context.Context, req *types.RequestOfferSnapshot) (*types.ResponseOfferSnapshot, error) {
	if app.OnOfferSnapshot != nil {
		return app.OnOfferSnapshot(ctx, req)
	}
	return &types.ResponseOfferSnapshot{}, nil
}

func (app *App) LoadSnapshotChunk(ctx context.Context, req *types.RequestLoadSnapshotChunk) (*types.ResponseLoadSnapshotChunk, error) {
	if app.OnLoadSnapshotChunk != nil {
		return app.OnLoadSnapshotChunk(ctx, req)
	}
	return &types.ResponseLoadSnapshotChunk{}, nil
}

func (app *App) ApplySnapshotChunk(_ context.Context, req *types.RequestApplySnapshotChunk) (*types.ResponseApplySnapshotChunk, error) {
	if app.OnApplySnapshotChunk != nil {
		return app.OnApplySnapshotChunk(app.ctx, req)
	}
	return &types.ResponseApplySnapshotChunk{}, nil
}

func AddCommonCheckTxEvents(resp *types.ResponseCheckTx, tx Tx) *types.ResponseCheckTx {
	resp.Events = getBaseTxEvents(tx)
	return resp
}

func getBaseTxEvents(tx Tx) []types.Event {
	base := []types.Event{
		{
			Type: "tx",
			Attributes: []types.EventAttribute{
				{
					Key:   "submitter",
					Value: tx.PubKeyHex(),
					Index: true,
				},
			},
		},
		{
			Type: "command",
			Attributes: []types.EventAttribute{
				{
					Key:   "type",
					Value: tx.Command().String(),
					Index: true,
				},
			},
		},
	}

	commandAttributes := []types.EventAttribute{}

	cmd := tx.GetCmd()
	if cmd == nil {
		return base
	}

	var market string
	if m, ok := cmd.(interface{ GetMarketId() string }); ok {
		market = m.GetMarketId()
	}
	if m, ok := cmd.(interface{ GetMarket() string }); ok {
		market = m.GetMarket()
	}
	if len(market) > 0 {
		commandAttributes = append(commandAttributes, types.EventAttribute{
			Key:   "market",
			Value: market,
			Index: true,
		})
	}

	var asset string
	if m, ok := cmd.(interface{ GetAssetId() string }); ok {
		asset = m.GetAssetId()
	}
	if m, ok := cmd.(interface{ GetAsset() string }); ok {
		asset = m.GetAsset()
	}
	if len(asset) > 0 {
		commandAttributes = append(commandAttributes, types.EventAttribute{
			Key:   "asset",
			Value: asset,
			Index: true,
		})
	}

	var reference string
	if m, ok := cmd.(interface{ GetReference() string }); ok {
		reference = m.GetReference()
	}
	if len(reference) > 0 {
		commandAttributes = append(commandAttributes, types.EventAttribute{
			Key:   "reference",
			Value: reference,
			Index: true,
		})
	}

	var proposal string
	if m, ok := cmd.(interface{ GetProposalId() string }); ok {
		proposal = m.GetProposalId()
	}
	if len(proposal) > 0 {
		commandAttributes = append(commandAttributes, types.EventAttribute{
			Key:   "proposal",
			Value: proposal,
			Index: true,
		})
	}

	if len(commandAttributes) > 0 {
		base[1].Attributes = append(base[1].Attributes, commandAttributes...)
	}

	return base
}

type MaybePartialError interface {
	error
	IsPartial() bool
}
