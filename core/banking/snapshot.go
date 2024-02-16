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

package banking

import (
	"context"
	"math/big"
	"sort"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/proto"
	"code.vegaprotocol.io/vega/logging"
	checkpoint "code.vegaprotocol.io/vega/protos/vega/checkpoint/v1"
	snapshot "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"

	"github.com/emirpasic/gods/sets/treeset"
)

var (
	withdrawalsKey           = (&types.PayloadBankingWithdrawals{}).Key()
	depositsKey              = (&types.PayloadBankingDeposits{}).Key()
	seenKey                  = (&types.PayloadBankingSeen{}).Key()
	assetActionsKey          = (&types.PayloadBankingAssetActions{}).Key()
	recurringTransfersKey    = (&types.PayloadBankingRecurringTransfers{}).Key()
	scheduledTransfersKey    = (&types.PayloadBankingScheduledTransfers{}).Key()
	bridgeStateKey           = (&types.PayloadBankingBridgeState{}).Key()
	recurringGovTransfersKey = (&types.PayloadBankingRecurringGovernanceTransfers{}).Key()
	scheduledGovTransfersKey = (&types.PayloadBankingScheduledGovernanceTransfers{}).Key()
	transferFeeDiscountsKey  = (&types.PayloadBankingTransferFeeDiscounts{}).Key()

	hashKeys = []string{
		withdrawalsKey,
		depositsKey,
		seenKey,
		assetActionsKey,
		recurringTransfersKey,
		scheduledTransfersKey,
		bridgeStateKey,
		recurringGovTransfersKey,
		scheduledGovTransfersKey,
		transferFeeDiscountsKey,
	}
)

type bankingSnapshotState struct {
	serialisedWithdrawals           []byte
	serialisedDeposits              []byte
	serialisedSeen                  []byte
	serialisedAssetActions          []byte
	serialisedRecurringTransfers    []byte
	serialisedScheduledTransfers    []byte
	serialisedBridgeState           []byte
	serialisedGovRecurringTransfers []byte
	serialisedGovScheduledTransfers []byte
	serialisedTransferFeeDiscounts  []byte
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
		Data: &types.PayloadBankingRecurringTransfers{
			BankingRecurringTransfers: e.getRecurringTransfers(),
		},
	}

	return proto.Marshal(payload.IntoProto())
}

func (e *Engine) serialiseScheduledTransfers() ([]byte, error) {
	payload := types.Payload{
		Data: &types.PayloadBankingScheduledTransfers{
			BankingScheduledTransfers: e.getScheduledTransfers(),
		},
	}

	return proto.Marshal(payload.IntoProto())
}

func (e *Engine) serialiseRecurringGovernanceTransfers() ([]byte, error) {
	payload := types.Payload{
		Data: &types.PayloadBankingRecurringGovernanceTransfers{
			BankingRecurringGovernanceTransfers: e.getRecurringGovernanceTransfers(),
		},
	}

	return proto.Marshal(payload.IntoProto())
}

func (e *Engine) serialiseScheduledGovernanceTransfers() ([]byte, error) {
	payload := types.Payload{
		Data: &types.PayloadBankingScheduledGovernanceTransfers{
			BankingScheduledGovernanceTransfers: e.getScheduledGovernanceTransfers(),
		},
	}

	return proto.Marshal(payload.IntoProto())
}

func (e *Engine) serialisedTransferFeeDiscounts() ([]byte, error) {
	partyAssetDiscounts := make([]*snapshot.PartyAssetAmount, 0, len(e.feeDiscountPerPartyAndAsset))

	for k, v := range e.feeDiscountPerPartyAndAsset {
		partyAssetDiscounts = append(partyAssetDiscounts, &snapshot.PartyAssetAmount{
			Party:  k.party,
			Asset:  k.asset,
			Amount: v.String(),
		})
	}
	sort.SliceStable(partyAssetDiscounts, func(i, j int) bool {
		if partyAssetDiscounts[i].Party == partyAssetDiscounts[j].Party {
			return partyAssetDiscounts[i].Asset < partyAssetDiscounts[j].Asset
		}
		return partyAssetDiscounts[i].Party < partyAssetDiscounts[j].Party
	})

	payload := types.Payload{
		Data: &types.PayloadBankingTransferFeeDiscounts{
			BankingTransferFeeDiscounts: &snapshot.BankingTransferFeeDiscounts{
				PartyAssetDiscount: partyAssetDiscounts,
			},
		},
	}

	return proto.Marshal(payload.IntoProto())
}

func (e *Engine) serialiseAssetActions() ([]byte, error) {
	payload := types.Payload{
		Data: &types.PayloadBankingAssetActions{
			BankingAssetActions: &types.BankingAssetActions{
				AssetAction: e.getAssetActions(),
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
	seen := &types.PayloadBankingSeen{
		BankingSeen: &types.BankingSeen{
			LastSeenEthBlock: e.lastSeenEthBlock,
		},
	}
	seen.BankingSeen.Refs = make([]string, 0, e.seenAssetActions.Size())
	iter := e.seenAssetActions.Iterator()
	for iter.Next() {
		seen.BankingSeen.Refs = append(seen.BankingSeen.Refs, iter.Value().(string))
	}
	payload := types.Payload{Data: seen}
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

func (e *Engine) serialiseK(serialFunc func() ([]byte, error), dataField *[]byte) ([]byte, error) {
	data, err := serialFunc()
	if err != nil {
		return nil, err
	}
	*dataField = data
	return data, nil
}

// get the serialised form and hash of the given key.
func (e *Engine) serialise(k string) ([]byte, error) {
	switch k {
	case depositsKey:
		return e.serialiseK(e.serialiseDeposits, &e.bss.serialisedDeposits)
	case withdrawalsKey:
		return e.serialiseK(e.serialiseWithdrawals, &e.bss.serialisedWithdrawals)
	case seenKey:
		return e.serialiseK(e.serialiseSeen, &e.bss.serialisedSeen)
	case assetActionsKey:
		return e.serialiseK(e.serialiseAssetActions, &e.bss.serialisedAssetActions)
	case recurringTransfersKey:
		return e.serialiseK(e.serialiseRecurringTransfers, &e.bss.serialisedRecurringTransfers)
	case scheduledTransfersKey:
		return e.serialiseK(e.serialiseScheduledTransfers, &e.bss.serialisedScheduledTransfers)
	case recurringGovTransfersKey:
		return e.serialiseK(e.serialiseRecurringGovernanceTransfers, &e.bss.serialisedGovRecurringTransfers)
	case scheduledGovTransfersKey:
		return e.serialiseK(e.serialiseScheduledGovernanceTransfers, &e.bss.serialisedGovScheduledTransfers)
	case transferFeeDiscountsKey:
		return e.serialiseK(e.serialisedTransferFeeDiscounts, &e.bss.serialisedTransferFeeDiscounts)
	case bridgeStateKey:
		return e.serialiseK(e.serialiseBridgeState, &e.bss.serialisedBridgeState)
	default:
		return nil, types.ErrSnapshotKeyDoesNotExist
	}
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
		return nil, e.restoreDeposits(pl.BankingDeposits, p)
	case *types.PayloadBankingWithdrawals:
		return nil, e.restoreWithdrawals(pl.BankingWithdrawals, p)
	case *types.PayloadBankingSeen:
		return nil, e.restoreSeen(pl.BankingSeen, p)
	case *types.PayloadBankingAssetActions:
		return nil, e.restoreAssetActions(pl.BankingAssetActions, p)
	case *types.PayloadBankingRecurringTransfers:
		return nil, e.restoreRecurringTransfers(ctx, pl.BankingRecurringTransfers, p)
	case *types.PayloadBankingScheduledTransfers:
		return nil, e.restoreScheduledTransfers(ctx, pl.BankingScheduledTransfers, p)
	case *types.PayloadBankingRecurringGovernanceTransfers:
		return nil, e.restoreRecurringGovernanceTransfers(ctx, pl.BankingRecurringGovernanceTransfers, p)
	case *types.PayloadBankingScheduledGovernanceTransfers:
		return nil, e.restoreScheduledGovernanceTransfers(ctx, pl.BankingScheduledGovernanceTransfers, p)
	case *types.PayloadBankingBridgeState:
		return nil, e.restoreBridgeState(pl.BankingBridgeState, p)
	case *types.PayloadBankingTransferFeeDiscounts:
		return nil, e.restoreTransferFeeDiscounts(pl.BankingTransferFeeDiscounts, p)
	default:
		return nil, types.ErrUnknownSnapshotType
	}
}

func (e *Engine) restoreRecurringTransfers(ctx context.Context, transfers *checkpoint.RecurringTransfers, p *types.Payload) error {
	var err error
	// ignore events here as we don't need to send them
	_ = e.loadRecurringTransfers(ctx, transfers)
	e.bss.serialisedRecurringTransfers, err = proto.Marshal(p.IntoProto())

	return err
}

func (e *Engine) restoreRecurringGovernanceTransfers(ctx context.Context, transfers []*checkpoint.GovernanceTransfer, p *types.Payload) error {
	var err error
	_ = e.loadRecurringGovernanceTransfers(ctx, transfers)
	e.bss.serialisedGovRecurringTransfers, err = proto.Marshal(p.IntoProto())
	return err
}

func (e *Engine) restoreScheduledTransfers(ctx context.Context, transfers []*checkpoint.ScheduledTransferAtTime, p *types.Payload) error {
	var err error

	// ignore events
	_, err = e.loadScheduledTransfers(ctx, transfers)
	if err != nil {
		return err
	}
	e.bss.serialisedScheduledTransfers, err = proto.Marshal(p.IntoProto())
	return err
}

func (e *Engine) restoreScheduledGovernanceTransfers(ctx context.Context, transfers []*checkpoint.ScheduledGovernanceTransferAtTime, p *types.Payload) error {
	var err error
	e.loadScheduledGovernanceTransfers(ctx, transfers)
	e.bss.serialisedGovScheduledTransfers, err = proto.Marshal(p.IntoProto())
	return err
}

func (e *Engine) restoreBridgeState(state *types.BankingBridgeState, p *types.Payload) (err error) {
	if state != nil {
		e.bridgeState = &bridgeState{
			active:   state.Active,
			block:    state.BlockHeight,
			logIndex: state.LogIndex,
		}
	}

	e.bss.serialisedBridgeState, err = proto.Marshal(p.IntoProto())
	return
}

func (e *Engine) restoreDeposits(deposits *types.BankingDeposits, p *types.Payload) error {
	var err error

	for _, d := range deposits.Deposit {
		e.deposits[d.ID] = d.Deposit
	}

	e.bss.serialisedDeposits, err = proto.Marshal(p.IntoProto())
	return err
}

func (e *Engine) restoreWithdrawals(withdrawals *types.BankingWithdrawals, p *types.Payload) error {
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

	e.bss.serialisedWithdrawals, err = proto.Marshal(p.IntoProto())

	return err
}

func (e *Engine) restoreSeen(seen *types.BankingSeen, p *types.Payload) error {
	var err error
	e.log.Info("restoring seen", logging.Int("n", len(seen.Refs)))
	e.seenAssetActions = treeset.NewWithStringComparator()
	for _, v := range seen.Refs {
		e.seenAssetActions.Add(v)
	}
	e.lastSeenEthBlock = seen.LastSeenEthBlock
	e.bss.serialisedSeen, err = proto.Marshal(p.IntoProto())
	return err
}

func (e *Engine) restoreAssetActions(aa *types.BankingAssetActions, p *types.Payload) error {
	var err error

	e.loadAssetActions(aa.AssetAction)
	for _, aa := range e.assetActions {
		if err := e.witness.RestoreResource(aa, e.onCheckDone); err != nil {
			e.log.Panic("unable to restore witness resource", logging.String("id", aa.id), logging.Error(err))
		}
	}

	e.bss.serialisedAssetActions, err = proto.Marshal(p.IntoProto())
	return err
}

func (e *Engine) restoreTransferFeeDiscounts(
	state *snapshot.BankingTransferFeeDiscounts,
	p *types.Payload,
) (err error) {
	if state == nil {
		return nil
	}

	e.feeDiscountPerPartyAndAsset = make(map[partyAssetKey]*num.Uint, len(state.PartyAssetDiscount))
	for _, v := range state.PartyAssetDiscount {
		discount, _ := num.UintFromString(v.Amount, 10)
		e.feeDiscountPerPartyAndAsset[e.feeDiscountKey(v.Asset, v.Party)] = discount
	}

	e.bss.serialisedTransferFeeDiscounts, err = proto.Marshal(p.IntoProto())
	return
}

func (e *Engine) OnEpochRestore(ctx context.Context, ep types.Epoch) {
	e.log.Debug("epoch restoration notification received", logging.String("epoch", ep.String()))
	e.currentEpoch = ep.Seq
}

func (e *Engine) OnStateLoaded(ctx context.Context) error {
	if e.lastSeenEthBlock != 0 {
		e.log.Info("restoring collateral bridge starting block", logging.Uint64("block", e.lastSeenEthBlock))
		e.ethEventSource.UpdateCollateralStartingBlock(e.lastSeenEthBlock)
	}
	return nil
}
