package banking

import (
	"context"
	"time"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
)

func (e *Engine) WithdrawBuiltinAsset(
	ctx context.Context, id, party, assetID string, amount *num.Uint) error {
	// build the withdrawal type
	w, ref := e.newWithdrawal(id, party, assetID, amount, time.Time{}, nil)
	w.Status = types.WithdrawalStatusRejected // default
	e.withdrawals[w.ID] = withdrawalRef{w, ref}

	asset, err := e.assets.Get(assetID)
	if err != nil {
		e.broker.Send(events.NewWithdrawalEvent(ctx, *w))
		e.log.Error("unable to get asset by id",
			logging.AssetID(assetID),
			logging.Error(err))
		return err
	}

	if !asset.IsBuiltinAsset() {
		e.broker.Send(events.NewWithdrawalEvent(ctx, *w))
		return ErrWrongAssetTypeUsedInBuiltinAssetChainEvent
	}

	return e.finalizeWithdraw(ctx, w)
}

func (e *Engine) DepositBuiltinAsset(
	ctx context.Context, d *types.BuiltinAssetDeposit, id string, nonce uint64) error {
	now := e.currentTime
	dep := e.newDeposit(id, d.PartyID, d.VegaAssetID, d.Amount, "") // no hash
	e.broker.Send(events.NewDepositEvent(ctx, *dep))
	asset, err := e.assets.Get(d.VegaAssetID)
	if err != nil {
		dep.Status = types.DepositStatusCancelled
		e.broker.Send(events.NewDepositEvent(ctx, *dep))
		e.log.Error("unable to get asset by id",
			logging.AssetID(d.VegaAssetID),
			logging.Error(err))
		return err
	}
	if !asset.IsBuiltinAsset() {
		dep.Status = types.DepositStatusCancelled
		e.broker.Send(events.NewDepositEvent(ctx, *dep))
		return ErrWrongAssetTypeUsedInBuiltinAssetChainEvent
	}

	aa := &assetAction{
		id:       dep.ID,
		state:    pendingState,
		builtinD: d,
		asset:    asset,
	}
	e.assetActs[aa.id] = aa
	e.deposits[dep.ID] = dep
	return e.witness.StartCheck(aa, e.onCheckDone, now.Add(defaultValidationDuration))
}

func (e *Engine) EnableBuiltinAsset(ctx context.Context, assetID string) error {
	return e.finalizeAssetList(ctx, assetID)
}
