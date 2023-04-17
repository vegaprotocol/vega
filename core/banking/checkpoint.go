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

package banking

import (
	"context"
	"sort"
	"sync/atomic"

	"code.vegaprotocol.io/vega/core/assets"
	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/proto"
	"code.vegaprotocol.io/vega/logging"
	checkpoint "code.vegaprotocol.io/vega/protos/vega/checkpoint/v1"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
	"github.com/emirpasic/gods/sets/treeset"
)

func (e *Engine) Name() types.CheckpointName {
	return types.BankingCheckpoint
}

func (e *Engine) Checkpoint() ([]byte, error) {
	msg := &checkpoint.Banking{
		TransfersAtTime:    e.getScheduledTransfers(),
		RecurringTransfers: e.getRecurringTransfers(),
		BridgeState:        e.getBridgeState(),
		LastSeenEthBlock:   e.lastSeenEthBlock,
	}

	msg.SeenRefs = make([]string, 0, e.seen.Size())
	iter := e.seen.Iterator()
	for iter.Next() {
		msg.SeenRefs = append(msg.SeenRefs, iter.Value().(string))
	}

	msg.AssetActions = make([]*checkpoint.AssetAction, 0, len(e.assetActs))
	for _, aa := range e.getAssetActions() {
		msg.AssetActions = append(msg.AssetActions, aa.IntoProto())
	}

	ret, err := proto.Marshal(msg)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (e *Engine) Load(ctx context.Context, data []byte) error {
	b := checkpoint.Banking{}
	if err := proto.Unmarshal(data, &b); err != nil {
		return err
	}

	evts, err := e.loadScheduledTransfers(ctx, b.TransfersAtTime)
	if err != nil {
		return err
	}

	evts = append(evts, e.loadRecurringTransfers(ctx, b.RecurringTransfers)...)

	e.loadBridgeState(b.BridgeState)

	e.seen = treeset.NewWithStringComparator()
	for _, v := range b.SeenRefs {
		e.seen.Add(v)
	}

	e.lastSeenEthBlock = b.LastSeenEthBlock
	if e.lastSeenEthBlock != 0 {
		e.log.Info("restoring collateral bridge starting block", logging.Uint64("block", e.lastSeenEthBlock))
		e.ethEventSource.UpdateCollateralStartingBlock(e.lastSeenEthBlock)
	}

	aa := make([]*types.AssetAction, 0, len(b.AssetActions))
	for _, a := range b.AssetActions {
		aa = append(aa, types.AssetActionFromProto(a))
	}
	e.loadAssetActions(aa)
	for _, aa := range e.assetActs {
		e.witness.StartCheck(aa, e.onCheckDone, e.timeService.GetTimeNow().Add(defaultValidationDuration))
	}

	if len(evts) > 0 {
		e.broker.SendBatch(evts)
	}

	return nil
}

func (e *Engine) loadAssetActions(aa []*types.AssetAction) {
	for _, v := range aa {
		var (
			err           error
			asset         *assets.Asset
			bridgeStopped *types.ERC20EventBridgeStopped
			bridgeResumed *types.ERC20EventBridgeResumed
		)

		// only others action than bridge stop and resume
		// have an actual asset associated
		if !v.BridgeResume && !v.BridgeStopped {
			asset, err = e.assets.Get(v.Asset)
			if err != nil {
				e.log.Panic("trying to restore an assetAction with no asset", logging.String("asset", v.Asset))
			}
		}

		if v.BridgeStopped {
			bridgeStopped = &types.ERC20EventBridgeStopped{BridgeStopped: true}
		}

		if v.BridgeResume {
			bridgeResumed = &types.ERC20EventBridgeResumed{BridgeResumed: true}
		}

		state := &atomic.Uint32{}
		state.Store(v.State)
		aa := &assetAction{
			id:                      v.ID,
			state:                   state,
			blockHeight:             v.BlockNumber,
			asset:                   asset,
			logIndex:                v.TxIndex,
			txHash:                  v.Hash,
			builtinD:                v.BuiltinD,
			erc20AL:                 v.Erc20AL,
			erc20D:                  v.Erc20D,
			erc20AssetLimitsUpdated: v.ERC20AssetLimitsUpdated,
			erc20BridgeStopped:      bridgeStopped,
			erc20BridgeResumed:      bridgeResumed,
			// this is needed every time now
			bridgeView: e.bridgeView,
		}

		if len(aa.getRef().Hash) == 0 {
			// if we're here it means that the IntoProto code has not done its job properly for a particular asset action type
			e.log.Panic("asset action has not been serialised correct and is empty", logging.String("txHash", aa.txHash))
		}

		e.assetActs[v.ID] = aa
		// store the deposit in the deposits
		if v.BuiltinD != nil {
			e.deposits[v.ID] = e.newDeposit(v.ID, v.BuiltinD.PartyID, v.BuiltinD.VegaAssetID, v.BuiltinD.Amount, v.Hash)
		} else if v.Erc20D != nil {
			e.deposits[v.ID] = e.newDeposit(v.ID, v.Erc20D.TargetPartyID, v.Erc20D.VegaAssetID, v.Erc20D.Amount, v.Hash)
		}
	}
}

func (e *Engine) loadBridgeState(state *checkpoint.BridgeState) {
	// this would eventually be nil if we restore from a checkpoint
	// which have been produce from an old version of the core.
	// we set it to active by default in the case
	if state == nil {
		e.bridgeState = &bridgeState{
			active: true,
		}
		return
	}

	e.bridgeState = &bridgeState{
		active:   state.Active,
		block:    state.BlockHeight,
		logIndex: state.LogIndex,
	}
}

func (e *Engine) loadScheduledTransfers(
	ctx context.Context, r []*checkpoint.ScheduledTransferAtTime,
) ([]events.Event, error) {
	evts := []events.Event{}
	for _, v := range r {
		transfers := make([]scheduledTransfer, 0, len(v.Transfers))
		for _, v := range v.Transfers {
			transfer, err := scheduledTransferFromProto(v)
			if err != nil {
				return nil, err
			}
			evts = append(evts, events.NewOneOffTransferFundsEvent(ctx, transfer.oneoff))
			transfers = append(transfers, transfer)
		}
		e.scheduledTransfers[v.DeliverOn] = transfers
	}

	return evts, nil
}

func (e *Engine) loadRecurringTransfers(
	ctx context.Context, r *checkpoint.RecurringTransfers,
) []events.Event {
	evts := []events.Event{}
	for _, v := range r.RecurringTransfers {
		transfer := types.RecurringTransferFromEvent(v)
		e.recurringTransfers = append(e.recurringTransfers, transfer)
		e.recurringTransfersMap[transfer.ID] = transfer
		evts = append(evts, events.NewRecurringTransferFundsEvent(ctx, transfer))
	}
	return evts
}

func (e *Engine) getBridgeState() *checkpoint.BridgeState {
	return &checkpoint.BridgeState{
		Active:      e.bridgeState.active,
		BlockHeight: e.bridgeState.block,
		LogIndex:    e.bridgeState.logIndex,
	}
}

func (e *Engine) getRecurringTransfers() *checkpoint.RecurringTransfers {
	out := &checkpoint.RecurringTransfers{
		RecurringTransfers: make([]*eventspb.Transfer, 0, len(e.recurringTransfers)),
	}

	for _, v := range e.recurringTransfers {
		out.RecurringTransfers = append(out.RecurringTransfers, v.IntoEvent(nil))
	}

	return out
}

func (e *Engine) getScheduledTransfers() []*checkpoint.ScheduledTransferAtTime {
	out := make([]*checkpoint.ScheduledTransferAtTime, 0, len(e.scheduledTransfers))

	for k, v := range e.scheduledTransfers {
		transfers := make([]*checkpoint.ScheduledTransfer, 0, len(v))
		for _, v := range v {
			transfers = append(transfers, v.ToProto())
		}
		out = append(out, &checkpoint.ScheduledTransferAtTime{DeliverOn: k, Transfers: transfers})
	}

	sort.SliceStable(out, func(i, j int) bool { return out[i].DeliverOn < out[j].DeliverOn })

	return out
}

func (e *Engine) getAssetActions() []*types.AssetAction {
	aa := make([]*types.AssetAction, 0, len(e.assetActs))
	for _, v := range e.assetActs {
		// this is optional as bridge action don't have one
		var assetID string
		if v.asset != nil {
			assetID = v.asset.ToAssetType().ID
		}

		var bridgeStopped bool
		if v.erc20BridgeStopped != nil {
			bridgeStopped = true
		}

		var bridgeResumed bool
		if v.erc20BridgeResumed != nil {
			bridgeResumed = true
		}

		aa = append(aa, &types.AssetAction{
			ID:                      v.id,
			State:                   v.state.Load(),
			BlockNumber:             v.blockHeight,
			Asset:                   assetID,
			TxIndex:                 v.logIndex,
			Hash:                    v.txHash,
			BuiltinD:                v.builtinD,
			Erc20AL:                 v.erc20AL,
			Erc20D:                  v.erc20D,
			ERC20AssetLimitsUpdated: v.erc20AssetLimitsUpdated,
			BridgeStopped:           bridgeStopped,
			BridgeResume:            bridgeResumed,
		})
	}

	sort.SliceStable(aa, func(i, j int) bool { return aa[i].ID < aa[j].ID })
	return aa
}
