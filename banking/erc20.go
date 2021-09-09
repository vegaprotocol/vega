package banking

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"code.vegaprotocol.io/vega/assets/erc20"
	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
)

var (
	ErrInvalidWithdrawalReferenceNonce = errors.New("invalid withdrawal reference nonce")
)

func (e *Engine) DepositERC20(
	ctx context.Context,
	d *types.ERC20Deposit,
	id string,
	blockNumber, txIndex uint64,
	txHash string,
) error {
	dep := e.newDeposit(id, d.TargetPartyID, d.VegaAssetID, d.Amount, txHash)

	// check if the asset is correct
	asset, err := e.assets.Get(d.VegaAssetID)
	if err != nil {
		dep.Status = types.DepositStatusCancelled
		e.broker.Send(events.NewDepositEvent(ctx, *dep))
		e.log.Error("unable to get asset by id",
			logging.AssetID(d.VegaAssetID),
			logging.Error(err))
		return err
	}

	if !asset.IsERC20() {
		dep.Status = types.DepositStatusCancelled
		e.broker.Send(events.NewDepositEvent(ctx, *dep))
		return fmt.Errorf("%v: %w", asset.String(), ErrWrongAssetTypeUsedInERC20ChainEvent)
	}

	aa := &assetAction{
		id:          dep.ID,
		state:       pendingState,
		erc20D:      d,
		asset:       asset,
		blockNumber: blockNumber,
		txIndex:     txIndex,
		hash:        txHash,
	}
	e.assetActs[aa.id] = aa
	e.deposits[dep.ID] = dep

	e.broker.Send(events.NewDepositEvent(ctx, *dep))
	return e.witness.StartCheck(aa, e.onCheckDone, e.currentTime.Add(defaultValidationDuration))
}

func (e *Engine) ERC20WithdrawalEvent(
	ctx context.Context, w *types.ERC20Withdrawal,
	blockNumber, txIndex uint64,
	txHash string,
) error {
	// check straight away if the withdrawal is signed
	nonce, ok := new(big.Int).SetString(w.ReferenceNonce, 10)
	if !ok {
		return fmt.Errorf("%s: %w", w.ReferenceNonce, ErrInvalidWithdrawalReferenceNonce)
	}

	withd, err := e.getWithdrawalFromRef(nonce)
	if err != nil {
		return fmt.Errorf("%s: %w", w.ReferenceNonce, err)
	}
	if withd.Status != types.WithdrawalStatusFinalized {
		return fmt.Errorf("%s: %w", withd.ID, ErrInvalidWithdrawalState)
	}
	if _, ok := e.notary.IsSigned(ctx, withd.ID, types.NodeSignatureKindAssetWithdrawal); !ok {
		return ErrWithdrawalNotReady
	}

	withd.WithdrawalDate = e.currentTime.UnixNano()
	withd.TxHash = txHash
	e.broker.Send(events.NewWithdrawalEvent(ctx, *withd))

	return nil
}

func (e *Engine) WithdrawERC20(
	ctx context.Context,
	id, party, assetID string,
	amount *num.Uint,
	ext *types.Erc20WithdrawExt,
) error {
	wext := &types.WithdrawExt{
		Ext: &types.WithdrawExt_Erc20{
			Erc20: ext,
		},
	}

	expiry := e.currentTime.Add(withdrawalsDefaultExpiry)
	w, ref := e.newWithdrawal(id, party, assetID, amount, expiry, wext)
	e.broker.Send(events.NewWithdrawalEvent(ctx, *w))
	e.withdrawals[w.ID] = withdrawalRef{w, ref}

	asset, err := e.assets.Get(assetID)
	if err != nil {
		w.Status = types.WithdrawalStatusCancelled
		e.broker.Send(events.NewWithdrawalEvent(ctx, *w))
		e.log.Debug("unable to get asset by id",
			logging.AssetID(assetID),
			logging.Error(err))
		return err
	}

	if !asset.IsERC20() {
		w.Status = types.WithdrawalStatusCancelled
		e.broker.Send(events.NewWithdrawalEvent(ctx, *w))
		return ErrWrongAssetUsedForERC20Withdraw
	}

	// try to withdraw if no error, this'll just abort
	if err := e.withdraw(ctx, w); err != nil {
		return err
	}

	// no check error as we checked earlier we had an erc20 asset.
	erc20asset, _ := asset.ERC20()

	// startup aggregating signature for the bundle
	return e.startERC20Signatures(ctx, w, erc20asset, ref)
}

func (e *Engine) startERC20Signatures(
	ctx context.Context,
	w *types.Withdrawal,
	asset *erc20.ERC20,
	ref *big.Int,
) error {
	// we were able to lock the funds, then we can send the vote through the network
	e.notary.StartAggregate(w.ID, types.NodeSignatureKindAssetWithdrawal)

	// if not a validator, we're good to go.
	if !e.top.IsValidator() {
		return nil
	}

	_, sig, err := asset.SignWithdrawal(
		w.Amount, w.ExpirationDate, w.Ext.GetErc20().GetReceiverAddress(), ref)
	if err != nil {
		// we don't cancel it here
		// we may not be able to sign for some reason, but other may be able
		// and we would aggregate enough signature
		e.log.Error("unable to sign withdrawal",
			logging.WithdrawalID(w.ID),
			logging.PartyID(w.PartyID),
			logging.AssetID(w.Asset),
			logging.BigUint("amount", w.Amount),
			logging.Error(err))
		return err
	}

	err = e.notary.SendSignature(
		ctx, w.ID, sig, types.NodeSignatureKindAssetWithdrawal)
	if err != nil {
		// we don't cancel it here
		// we may not be able to sign for some reason, but other may be able
		// and we would aggregate enough signature
		e.log.Error("unable to send node signature",
			logging.WithdrawalID(w.ID),
			logging.PartyID(w.PartyID),
			logging.AssetID(w.Asset),
			logging.BigUint("amount", w.Amount),
			logging.Error(err))
		return err
	}

	return nil
}
