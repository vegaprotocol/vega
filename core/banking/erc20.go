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
	"errors"
	"fmt"
	"math/big"
	"time"

	"code.vegaprotocol.io/vega/core/assets/erc20"
	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
)

var (
	ErrInvalidWithdrawalReferenceNonce       = errors.New("invalid withdrawal reference nonce")
	ErrAssetAlreadyBeingListed               = errors.New("asset already being listed")
	ErrWithdrawalDisabledWhenBridgeIsStopped = errors.New("withdrawal issuance is disabled when the erc20 is stopped")
)

type ERC20BridgeView interface {
	FindAssetList(al *types.ERC20AssetList, blockNumber, logIndex uint64) error
	FindBridgeStopped(al *types.ERC20EventBridgeStopped, blockNumber, logIndex uint64) error
	FindBridgeResumed(al *types.ERC20EventBridgeResumed, blockNumber, logIndex uint64) error
	FindDeposit(d *types.ERC20Deposit, blockNumber, logIndex uint64, ethAssetAddress string) error
	FindAssetLimitsUpdated(update *types.ERC20AssetLimitsUpdated, blockNumber uint64, logIndex uint64, ethAssetAddress string) error
}

func (e *Engine) EnableERC20(
	_ context.Context,
	al *types.ERC20AssetList,
	id string,
	blockNumber, txIndex uint64,
	txHash string,
) error {
	asset, _ := e.assets.Get(al.VegaAssetID)
	if _, ok := e.assetActs[al.VegaAssetID]; ok {
		e.log.Error("asset already being listed", logging.AssetID(al.VegaAssetID))
		return ErrAssetAlreadyBeingListed
	}

	aa := &assetAction{
		id:          id,
		state:       pendingState,
		erc20AL:     al,
		asset:       asset,
		blockHeight: blockNumber,
		logIndex:    txIndex,
		txHash:      txHash,
		bridgeView:  e.bridgeView,
	}
	e.assetActs[aa.id] = aa
	return e.witness.StartCheck(aa, e.onCheckDone, e.timeService.GetTimeNow().Add(defaultValidationDuration))
}

func (e *Engine) UpdateERC20(
	_ context.Context,
	event *types.ERC20AssetLimitsUpdated,
	id string,
	blockNumber, txIndex uint64,
	txHash string,
) error {
	asset, err := e.assets.Get(event.VegaAssetID)
	if err != nil {
		e.log.Panic("couldn't retrieve the ERC20 asset",
			logging.AssetID(event.VegaAssetID),
		)
	}

	aa := &assetAction{
		id:                      id,
		state:                   pendingState,
		erc20AssetLimitsUpdated: event,
		asset:                   asset,
		blockHeight:             blockNumber,
		logIndex:                txIndex,
		txHash:                  txHash,
		bridgeView:              e.bridgeView,
	}
	e.assetActs[aa.id] = aa
	return e.witness.StartCheck(aa, e.onCheckDone, e.timeService.GetTimeNow().Add(defaultValidationDuration))
}

func (e *Engine) DepositERC20(
	ctx context.Context,
	d *types.ERC20Deposit,
	id string,
	blockNumber, logIndex uint64,
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
		blockHeight: blockNumber,
		logIndex:    logIndex,
		txHash:      txHash,
		bridgeView:  e.bridgeView,
	}
	e.assetActs[aa.id] = aa
	e.deposits[dep.ID] = dep

	e.broker.Send(events.NewDepositEvent(ctx, *dep))
	return e.witness.StartCheck(aa, e.onCheckDone, e.timeService.GetTimeNow().Add(defaultValidationDuration))
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

	withd.WithdrawalDate = e.timeService.GetTimeNow().UnixNano()
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
	if e.bridgeState.IsStopped() {
		return ErrWithdrawalDisabledWhenBridgeIsStopped
	}

	wext := &types.WithdrawExt{
		Ext: &types.WithdrawExtErc20{
			Erc20: ext,
		},
	}

	w, ref := e.newWithdrawal(id, party, assetID, amount, wext)
	e.broker.Send(events.NewWithdrawalEvent(ctx, *w))
	e.withdrawals[w.ID] = withdrawalRef{w, ref}

	asset, err := e.assets.Get(assetID)
	if err != nil {
		w.Status = types.WithdrawalStatusRejected
		e.broker.Send(events.NewWithdrawalEvent(ctx, *w))
		e.log.Debug("unable to get asset by id",
			logging.AssetID(assetID),
			logging.Error(err))
		return err
	}

	if a, ok := asset.ERC20(); !ok {
		w.Status = types.WithdrawalStatusRejected
		e.broker.Send(events.NewWithdrawalEvent(ctx, *w))
		return ErrWrongAssetUsedForERC20Withdraw
	} else if threshold := a.Type().Details.GetERC20().WithdrawThreshold; threshold != nil && threshold.NEQ(num.UintZero()) {
		// a delay will be applied on this withdrawal
		if threshold.LT(amount) {
			e.log.Debug("withdraw threshold breached, delay will be applied",
				logging.PartyID(party),
				logging.BigUint("threshold", threshold),
				logging.BigUint("amount", amount),
				logging.AssetID(assetID),
				logging.Error(err))
		}
	}

	// try to withdraw if no error, this'll just abort
	if err := e.finalizeWithdraw(ctx, w); err != nil {
		return err
	}

	// no check error as we checked earlier we had an erc20 asset.
	erc20asset, _ := asset.ERC20()

	// startup aggregating signature for the bundle
	return e.startERC20Signatures(w, erc20asset, ref)
}

func (e *Engine) startERC20Signatures(w *types.Withdrawal, asset *erc20.ERC20, ref *big.Int) error {
	var (
		signature []byte
		err       error
	)

	creation := time.Unix(0, w.CreationDate)
	// if we are a validator, we want to build a signature
	if e.top.IsValidator() {
		_, signature, err = asset.SignWithdrawal(
			w.Amount, w.Ext.GetErc20().GetReceiverAddress(), ref, creation)
		if err != nil {
			// there's no reason we cannot build the signature here
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
		// there's no reason we cannot find the withdrawal here
		// apart if the node isn't configured properly
		e.log.Panic("unable to find withdrawal",
			logging.WithdrawalID(resource))
	}
	w := wref.w

	asset, err := e.assets.Get(w.Asset)
	if err != nil {
		// there's no reason we cannot build the signature here
		// apart if the node isn't configure properly
		e.log.Panic("unable to get asset when offering signature",
			logging.WithdrawalID(w.ID),
			logging.PartyID(w.PartyID),
			logging.AssetID(w.Asset),
			logging.BigUint("amount", w.Amount),
			logging.Error(err))
	}

	creation := time.Unix(0, w.CreationDate)
	erc20asset, _ := asset.ERC20()
	_, signature, err := erc20asset.SignWithdrawal(
		w.Amount, w.Ext.GetErc20().GetReceiverAddress(), wref.ref, creation)
	if err != nil {
		// there's no reason we cannot build the signature here
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
