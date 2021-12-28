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

var ErrInvalidWithdrawalReferenceNonce = errors.New("invalid withdrawal reference nonce")

func (e *Engine) EnableERC20(
	ctx context.Context,
	al *types.ERC20AssetList,
	id string,
	blockNumber, txIndex uint64,
	txHash string,
) error {
	asset, _ := e.assets.Get(al.VegaAssetID)
	aa := &assetAction{
		id:          id,
		state:       pendingState,
		erc20AL:     al,
		asset:       asset,
		blockNumber: blockNumber,
		txIndex:     txIndex,
		hash:        txHash,
	}
	e.assetActs[aa.id] = aa
	return e.witness.StartCheck(aa, e.onCheckDone, e.currentTime.Add(defaultValidationDuration))
}

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
	if err := e.finalizeWithdraw(ctx, w); err != nil {
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
	var (
		signature []byte
		err       error
	)

	// if we are a validator, we want to build a signature
	if e.top.IsValidator() {
		_, signature, err = asset.SignWithdrawal(
			w.Amount, w.Ext.GetErc20().GetReceiverAddress(), ref)
		if err != nil {
			// there's not reason we cannot build the signature here
			// apart if the node isn't configure properly
			e.log.Panic("unable to sign withdrawal",
				logging.WithdrawalID(w.ID),
				logging.PartyID(w.PartyID),
				logging.AssetID(w.Asset),
				logging.BigUint("amount", w.Amount),
				logging.Error(err))
		}
	}

	// we were able to lock the funds, then we can send the vote through the network
	e.notary.StartAggregate(w.ID, types.NodeSignatureKindAssetWithdrawal, signature)

	return nil
}

func (e *Engine) offerERC20NotarySignatures(resource string) []byte {
	if !e.top.IsValidator() {
		return nil
	}

	wref, ok := e.withdrawals[resource]
	if !ok {
		// there's not reason we cannot find the withdrawal here
		// apart if the node isn't configured properly
		e.log.Panic("unable to find withdrawal",
			logging.WithdrawalID(resource))
	}
	w := wref.w

	asset, err := e.assets.Get(w.Asset)
	if err != nil {
		// there's not reason we cannot build the signature here
		// apart if the node isn't configure properly
		e.log.Panic("unable to get asset when offering signature",
			logging.WithdrawalID(w.ID),
			logging.PartyID(w.PartyID),
			logging.AssetID(w.Asset),
			logging.BigUint("amount", w.Amount),
			logging.Error(err))
	}

	erc20asset, _ := asset.ERC20()
	_, signature, err := erc20asset.SignWithdrawal(
		w.Amount, w.Ext.GetErc20().GetReceiverAddress(), wref.ref)
	if err != nil {
		// there's not reason we cannot build the signature here
		// apart if the node isn't configure properly
		e.log.Panic("unable to sign withdrawal",
			logging.WithdrawalID(w.ID),
			logging.PartyID(w.PartyID),
			logging.AssetID(w.Asset),
			logging.BigUint("amount", w.Amount),
			logging.Error(err))
	}

	return signature
}
