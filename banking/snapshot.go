package banking

import (
	"context"
	"errors"
	"math/big"
	"sort"

	checkpoint "code.vegaprotocol.io/protos/vega/checkpoint/v1"
	"code.vegaprotocol.io/vega/libs/proto"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/types"
)

var (
	withdrawalsKey        = (&types.PayloadBankingWithdrawals{}).Key()
	depositsKey           = (&types.PayloadBankingDeposits{}).Key()
	seenKey               = (&types.PayloadBankingSeen{}).Key()
	assetActionsKey       = (&types.PayloadBankingAssetActions{}).Key()
	recurringTransfersKey = (&types.PayloadBankingRecurringTransfers{}).Key()
	scheduledTransfersKey = (&types.PayloadBankingScheduledTransfers{}).Key()

	hashKeys = []string{
		withdrawalsKey,
		depositsKey,
		seenKey,
		assetActionsKey,
		recurringTransfersKey,
		scheduledTransfersKey,
	}

	ErrSnapshotKeyDoesNotExist = errors.New("unknown key for banking snapshot")
)

type bankingSnapshotState struct {
	changedWithdrawals           bool
	changedDeposits              bool
	changedSeen                  bool
	changedAssetActions          bool
	changedRecurringTransfers    bool
	changedScheduledTransfers    bool
	serialisedWithdrawals        []byte
	serialisedDeposits           []byte
	serialisedSeen               []byte
	serialisedAssetActions       []byte
	serialisedRecurringTransfers []byte
	serialisedScheduledTransfers []byte
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

func (e *Engine) serialiseAssetActions() ([]byte, error) {
	aa := make([]*types.AssetAction, 0, len(e.assetActs))
	for _, v := range e.assetActs {
		aa = append(aa, &types.AssetAction{
			ID:          v.id,
			State:       v.state,
			BlockNumber: v.blockNumber,
			Asset:       v.asset.ToAssetType().ID,
			TxIndex:     v.txIndex,
			Hash:        v.hash,
			BuiltinD:    v.builtinD,
			Erc20AL:     v.erc20AL,
			Erc20D:      v.erc20D,
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
		return e.serialiseK(k, e.serialiseRecurringTransfers, &e.bss.serialisedRecurringTransfers, &e.bss.changedRecurringTransfers)
	case scheduledTransfersKey:
		return e.serialiseK(k, e.serialiseScheduledTransfers, &e.bss.serialisedScheduledTransfers, &e.bss.changedScheduledTransfers)
	default:
		return nil, types.ErrSnapshotKeyDoesNotExist
	}
}

func (e *Engine) HasChanged(k string) bool {
	switch k {
	case depositsKey:
		return e.bss.changedDeposits
	case withdrawalsKey:
		return e.bss.changedWithdrawals
	case seenKey:
		return e.bss.changedSeen
	case assetActionsKey:
		return e.bss.changedAssetActions
	case recurringTransfersKey:
		return e.bss.changedRecurringTransfers
	case scheduledTransfersKey:
		return e.bss.changedScheduledTransfers
	default:
		return false
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
		return nil, e.restoreDeposits(ctx, pl.BankingDeposits, p)
	case *types.PayloadBankingWithdrawals:
		return nil, e.restoreWithdrawals(ctx, pl.BankingWithdrawals, p)
	case *types.PayloadBankingSeen:
		return nil, e.restoreSeen(ctx, pl.BankingSeen, p)
	case *types.PayloadBankingAssetActions:
		return nil, e.restoreAssetActions(ctx, pl.BankingAssetActions, p)
	case *types.PayloadBankingRecurringTransfers:
		return nil, e.restoreRecurringTransfers(ctx, pl.BankingRecurringTransfers, p)
	case *types.PayloadBankingScheduledTransfers:
		return nil, e.restoreScheduledTransfers(ctx, pl.BankingScheduledTransfers, p)
	default:
		return nil, types.ErrUnknownSnapshotType
	}
}

func (e *Engine) restoreRecurringTransfers(ctx context.Context, transfers *checkpoint.RecurringTransfers, p *types.Payload) error {
	var err error
	// ignore events here as we don't need to send them
	_ = e.loadRecurringTransfers(ctx, transfers)
	e.bss.changedRecurringTransfers = false
	e.bss.serialisedRecurringTransfers, err = proto.Marshal(p.IntoProto())

	return err
}

func (e *Engine) restoreScheduledTransfers(ctx context.Context, transfers []*checkpoint.ScheduledTransferAtTime, p *types.Payload) error {
	var err error

	// ignore events
	_, err = e.loadScheduledTransfers(ctx, transfers)
	if err != nil {
		return err
	}
	e.bss.changedScheduledTransfers = false
	e.bss.serialisedScheduledTransfers, err = proto.Marshal(p.IntoProto())
	return err
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
		asset, err := e.assets.Get(v.Asset)
		if err != nil {
			e.log.Panic("trying to restore an assetAction with no asset", logging.String("asset", v.Asset))
		}

		aa := &assetAction{
			id:          v.ID,
			state:       v.State,
			blockNumber: v.BlockNumber,
			asset:       asset,
			txIndex:     v.TxIndex,
			hash:        v.Hash,
			builtinD:    v.BuiltinD,
			erc20AL:     v.Erc20AL,
			erc20D:      v.Erc20D,
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
