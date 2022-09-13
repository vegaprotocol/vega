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
	"math/big"
	"sort"

	"code.vegaprotocol.io/vega/core/assets"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/proto"
	"code.vegaprotocol.io/vega/logging"
	checkpoint "code.vegaprotocol.io/vega/protos/vega/checkpoint/v1"
)

var (
	withdrawalsKey        = (&types.PayloadBankingWithdrawals{}).Key()
	depositsKey           = (&types.PayloadBankingDeposits{}).Key()
	seenKey               = (&types.PayloadBankingSeen{}).Key()
	assetActionsKey       = (&types.PayloadBankingAssetActions{}).Key()
	recurringTransfersKey = (&types.PayloadBankingRecurringTransferInstructions{}).Key()
	scheduledTransfersKey = (&types.PayloadBankingScheduledTransferInstructions{}).Key()
	bridgeStateKey        = (&types.PayloadBankingBridgeState{}).Key()

	hashKeys = []string{
		withdrawalsKey,
		depositsKey,
		seenKey,
		assetActionsKey,
		recurringTransfersKey,
		scheduledTransfersKey,
		bridgeStateKey,
	}
)

type bankingSnapshotState struct {
	changedWithdrawals                      bool
	changedDeposits                         bool
	changedSeen                             bool
	changedAssetActions                     bool
	changedRecurringTransferInstructions    bool
	changedScheduledTransferInstructions    bool
	changedBridgeState                      bool
	serialisedWithdrawals                   []byte
	serialisedDeposits                      []byte
	serialisedSeen                          []byte
	serialisedAssetActions                  []byte
	serialisedRecurringTransferInstructions []byte
	serialisedScheduledTransferInstructions []byte
	serialisedBridgeState                   []byte
}

func (e *Engine) Namespace() types.SnapshotNamespace {
	return types.BankingSnapshot
}

func (e *Engine) Keys() []string {
	return hashKeys
}

func (e *Engine) Stopped() bool {
	return false
}

func (e *Engine) serialiseBridgeState() ([]byte, error) {
	payload := types.Payload{
		Data: &types.PayloadBankingBridgeState{
			BankingBridgeState: &types.BankingBridgeState{
				Active:      e.bridgeState.active,
				BlockHeight: e.bridgeState.block,
				LogIndex:    e.bridgeState.logIndex,
			},
		},
	}

	return proto.Marshal(payload.IntoProto())
}

func (e *Engine) serialiseRecurringTransfers() ([]byte, error) {
	payload := types.Payload{
		Data: &types.PayloadBankingRecurringTransferInstructions{
			BankingRecurringTransferInstructions: e.getRecurringTransferInstructions(),
		},
	}

	return proto.Marshal(payload.IntoProto())
}

func (e *Engine) serialiseScheduledTransferInstructions() ([]byte, error) {
	payload := types.Payload{
		Data: &types.PayloadBankingScheduledTransferInstructions{
			BankingScheduledTransferInstructions: e.getScheduledTransferInstructions(),
		},
	}

	return proto.Marshal(payload.IntoProto())
}

func (e *Engine) serialiseAssetActions() ([]byte, error) {
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
			State:                   v.state,
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

	payload := types.Payload{
		Data: &types.PayloadBankingAssetActions{
			BankingAssetActions: &types.BankingAssetActions{
				AssetAction: aa,
			},
		},
	}
	return proto.Marshal(payload.IntoProto())
}

func (e *Engine) serialiseWithdrawals() ([]byte, error) {
	withdrawals := make([]*types.RWithdrawal, 0, len(e.withdrawals))
	for _, v := range e.withdrawals {
		withdrawals = append(withdrawals, &types.RWithdrawal{Ref: v.ref.String(), Withdrawal: v.w})
	}

	sort.SliceStable(withdrawals, func(i, j int) bool { return withdrawals[i].Ref < withdrawals[j].Ref })

	payload := types.Payload{
		Data: &types.PayloadBankingWithdrawals{
			BankingWithdrawals: &types.BankingWithdrawals{
				Withdrawals: withdrawals,
			},
		},
	}
	return proto.Marshal(payload.IntoProto())
}

func (e *Engine) serialiseSeen() ([]byte, error) {
	payload := types.Payload{
		Data: &types.PayloadBankingSeen{
			BankingSeen: &types.BankingSeen{
				Refs: e.seenSlice,
			},
		},
	}

	return proto.Marshal(payload.IntoProto())
}

func (e *Engine) serialiseDeposits() ([]byte, error) {
	e.log.Debug("serialiseDeposits: called")
	deposits := make([]*types.BDeposit, 0, len(e.deposits))
	for _, v := range e.deposits {
		deposits = append(deposits, &types.BDeposit{ID: v.ID, Deposit: v})
	}

	sort.SliceStable(deposits, func(i, j int) bool { return deposits[i].ID < deposits[j].ID })

	if e.log.IsDebug() {
		e.log.Info("serialiseDeposits: number of deposits:", logging.Int("len(deposits)", len(deposits)))
		for i, d := range deposits {
			e.log.Info("serialiseDeposits:", logging.Int("index", i), logging.String("ID", d.ID), logging.String("deposit", d.Deposit.String()))
		}
	}
	payload := types.Payload{
		Data: &types.PayloadBankingDeposits{
			BankingDeposits: &types.BankingDeposits{
				Deposit: deposits,
			},
		},
	}

	return proto.Marshal(payload.IntoProto())
}

func (e *Engine) serialiseK(k string, serialFunc func() ([]byte, error), dataField *[]byte, changedField *bool) ([]byte, error) {
	if !e.HasChanged(k) {
		if dataField == nil {
			return nil, nil
		}
		return *dataField, nil
	}
	data, err := serialFunc()
	if err != nil {
		return nil, err
	}
	*dataField = data
	*changedField = false
	return data, nil
}

// get the serialised form and hash of the given key.
func (e *Engine) serialise(k string) ([]byte, error) {
	switch k {
	case depositsKey:
		return e.serialiseK(k, e.serialiseDeposits, &e.bss.serialisedDeposits, &e.bss.changedDeposits)
	case withdrawalsKey:
		return e.serialiseK(k, e.serialiseWithdrawals, &e.bss.serialisedWithdrawals, &e.bss.changedWithdrawals)
	case seenKey:
		return e.serialiseK(k, e.serialiseSeen, &e.bss.serialisedSeen, &e.bss.changedSeen)
	case assetActionsKey:
		return e.serialiseK(k, e.serialiseAssetActions, &e.bss.serialisedAssetActions, &e.bss.changedAssetActions)
	case recurringTransfersKey:
		return e.serialiseK(k, e.serialiseRecurringTransfers, &e.bss.serialisedRecurringTransferInstructions, &e.bss.changedRecurringTransferInstructions)
	case scheduledTransfersKey:
		return e.serialiseK(k, e.serialiseScheduledTransferInstructions, &e.bss.serialisedScheduledTransferInstructions, &e.bss.changedScheduledTransferInstructions)
	case bridgeStateKey:
		return e.serialiseK(k, e.serialiseBridgeState, &e.bss.serialisedBridgeState, &e.bss.changedBridgeState)
	default:
		return nil, types.ErrSnapshotKeyDoesNotExist
	}
}

func (e *Engine) HasChanged(k string) bool {
	// switch k {
	// case depositsKey:
	// 	return e.bss.changedDeposits
	// case withdrawalsKey:
	// 	return e.bss.changedWithdrawals
	// case seenKey:
	// 	return e.bss.changedSeen
	// case assetActionsKey:
	// 	return e.bss.changedAssetActions
	// case recurringTransfersKey:
	// 	return e.bss.changedRecurringTransfers
	// case scheduledTransfersKey:
	// 	return e.bss.changedScheduledTransfers
	// default:
	// 	return false
	// }
	return true
}

func (e *Engine) GetState(k string) ([]byte, []types.StateProvider, error) {
	state, err := e.serialise(k)
	return state, nil, err
}

func (e *Engine) LoadState(ctx context.Context, p *types.Payload) ([]types.StateProvider, error) {
	if e.Namespace() != p.Data.Namespace() {
		return nil, types.ErrInvalidSnapshotNamespace
	}
	// see what we're reloading
	switch pl := p.Data.(type) {
	case *types.PayloadBankingDeposits:
		return nil, e.restoreDeposits(ctx, pl.BankingDeposits, p)
	case *types.PayloadBankingWithdrawals:
		return nil, e.restoreWithdrawals(ctx, pl.BankingWithdrawals, p)
	case *types.PayloadBankingSeen:
		return nil, e.restoreSeen(ctx, pl.BankingSeen, p)
	case *types.PayloadBankingAssetActions:
		return nil, e.restoreAssetActions(ctx, pl.BankingAssetActions, p)
	case *types.PayloadBankingRecurringTransferInstructions:
		return nil, e.restoreRecurringTransferInstructions(ctx, pl.BankingRecurringTransferInstructions, p)
	case *types.PayloadBankingScheduledTransferInstructions:
		return nil, e.restoreScheduledTransferInstructions(ctx, pl.BankingScheduledTransferInstructions, p)
	case *types.PayloadBankingBridgeState:
		return nil, e.restoreBridgeState(ctx, pl.BankingBridgeState, p)
	default:
		return nil, types.ErrUnknownSnapshotType
	}
}

func (e *Engine) restoreRecurringTransferInstructions(ctx context.Context, transfers *checkpoint.RecurringTransferInstructions, p *types.Payload) error {
	var err error
	// ignore events here as we don't need to send them
	_ = e.loadRecurringTransferInstructions(ctx, transfers)
	e.bss.changedRecurringTransferInstructions = false
	e.bss.serialisedRecurringTransferInstructions, err = proto.Marshal(p.IntoProto())

	return err
}

func (e *Engine) restoreScheduledTransferInstructions(ctx context.Context, transfers []*checkpoint.ScheduledTransferInstructionAtTime, p *types.Payload) error {
	var err error

	// ignore events
	_, err = e.loadScheduledTransferInstructions(ctx, transfers)
	if err != nil {
		return err
	}
	e.bss.changedScheduledTransferInstructions = false
	e.bss.serialisedScheduledTransferInstructions, err = proto.Marshal(p.IntoProto())
	return err
}

func (e *Engine) restoreBridgeState(ctx context.Context, state *types.BankingBridgeState, p *types.Payload) (err error) {
	if state != nil {
		e.bridgeState = &bridgeState{
			active:   state.Active,
			block:    state.BlockHeight,
			logIndex: state.LogIndex,
		}
	}

	e.bss.changedBridgeState = false
	e.bss.serialisedBridgeState, err = proto.Marshal(p.IntoProto())
	return
}

func (e *Engine) restoreDeposits(ctx context.Context, deposits *types.BankingDeposits, p *types.Payload) error {
	var err error

	for _, d := range deposits.Deposit {
		e.deposits[d.ID] = d.Deposit
	}

	e.bss.serialisedDeposits, err = proto.Marshal(p.IntoProto())
	e.bss.changedDeposits = false
	return err
}

func (e *Engine) restoreWithdrawals(ctx context.Context, withdrawals *types.BankingWithdrawals, p *types.Payload) error {
	var err error
	for _, w := range withdrawals.Withdrawals {
		ref := new(big.Int)
		ref.SetString(w.Ref, 10)
		e.withdrawalCnt.Add(e.withdrawalCnt, big.NewInt(1))
		e.withdrawals[w.Withdrawal.ID] = withdrawalRef{
			w:   w.Withdrawal,
			ref: ref,
		}
	}

	e.bss.changedWithdrawals = false
	e.bss.serialisedWithdrawals, err = proto.Marshal(p.IntoProto())

	return err
}

func (e *Engine) restoreSeen(ctx context.Context, seen *types.BankingSeen, p *types.Payload) error {
	var err error
	e.log.Info("restoring seen", logging.Int("n", len(seen.Refs)))
	for _, s := range seen.Refs {
		e.seen[s] = struct{}{}
	}
	e.seenSlice = seen.Refs
	e.bss.changedSeen = false
	e.bss.serialisedSeen, err = proto.Marshal(p.IntoProto())
	return err
}

func (e *Engine) restoreAssetActions(ctx context.Context, aa *types.BankingAssetActions, p *types.Payload) error {
	var err error
	for _, v := range aa.AssetAction {
		var (
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

		aa := &assetAction{
			id:                      v.ID,
			state:                   v.State,
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
		if err := e.witness.RestoreResource(aa, e.onCheckDone); err != nil {
			e.log.Panic("unable to restore witness resource", logging.String("id", v.ID), logging.Error(err))
		}
	}

	e.bss.changedAssetActions = false
	e.bss.serialisedAssetActions, err = proto.Marshal(p.IntoProto())
	return err
}

func (e *Engine) OnEpochRestore(ctx context.Context, ep types.Epoch) {
	e.log.Debug("epoch restoration notification received", logging.String("epoch", ep.String()))
	e.currentEpoch = ep.Seq
}
