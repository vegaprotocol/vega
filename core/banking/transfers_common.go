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
	"fmt"
	"time"

	"code.vegaprotocol.io/vega/core/assets"
	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
)

func (e *Engine) OnRewardsUpdateFrequencyUpdate(ctx context.Context, d time.Duration) error {
	if !e.nextMetricUpdate.IsZero() {
		e.nextMetricUpdate = e.nextMetricUpdate.Add(-e.metricUpdateFrequency)
	}
	e.nextMetricUpdate = e.nextMetricUpdate.Add(d)
	e.metricUpdateFrequency = d
	return nil
}

func (e *Engine) OnTransferFeeFactorUpdate(ctx context.Context, f num.Decimal) error {
	e.transferFeeFactor = f
	return nil
}

func (e *Engine) OnMinTransferQuantumMultiple(ctx context.Context, f num.Decimal) error {
	e.minTransferQuantumMultiple = f
	return nil
}

func (e *Engine) OnMaxQuantumAmountUpdate(ctx context.Context, f num.Decimal) error {
	e.maxQuantumAmount = f
	return nil
}

func (e *Engine) OnTransferFeeDiscountDecayFractionUpdate(ctx context.Context, v num.Decimal) error {
	e.feeDiscountDecayFraction = v
	return nil
}

func (e *Engine) OnTransferFeeDiscountMinimumTrackedAmountUpdate(ctx context.Context, v num.Decimal) error {
	e.feeDiscountMinimumTrackedAmount = v
	return nil
}

func (e *Engine) TransferFunds(
	ctx context.Context,
	transfer *types.TransferFunds,
) error {
	now := e.timeService.GetTimeNow()
	// add timestamps straight away
	switch transfer.Kind {
	case types.TransferCommandKindOneOff:
		transfer.OneOff.Timestamp = now
		return e.oneOffTransfer(ctx, transfer.OneOff)
	case types.TransferCommandKindRecurring:
		transfer.Recurring.Timestamp = now
		return e.recurringTransfer(ctx, transfer.Recurring)
	default:
		return ErrUnsupportedTransferKind
	}
}

func (e *Engine) CheckTransfer(t *types.TransferBase) error {
	// ensure asset exists
	a, err := e.assets.Get(t.Asset)
	if err != nil {
		e.log.Debug("cannot transfer funds, invalid asset", logging.Error(err))
		return fmt.Errorf("could not transfer funds, %w", err)
	}

	if err = t.IsValid(); err != nil {
		return fmt.Errorf("could not transfer funds, %w", err)
	}

	if err := e.ensureMinimalTransferAmount(a, t.Amount, t.FromAccountType, t.From, t.FromDerivedKey); err != nil {
		return err
	}

	if err = e.ensureFeeForTransferFunds(a, t.Amount, t.From, t.FromAccountType, t.FromDerivedKey, t.To); err != nil {
		return fmt.Errorf("could not transfer funds, %w", err)
	}
	return nil
}

func (e *Engine) ensureMinimalTransferAmount(
	a *assets.Asset,
	amount *num.Uint,
	fromAccType types.AccountType,
	from string,
	fromSubAccount *string,
) error {
	quantum := a.Type().Details.Quantum
	// no reason this would produce an error
	minAmount, _ := num.UintFromDecimal(quantum.Mul(e.minTransferQuantumMultiple))

	// no verify amount
	if amount.LT(minAmount) {
		if fromAccType == types.AccountTypeVestedRewards {
			if fromSubAccount != nil {
				from = *fromSubAccount
			}
			return e.ensureMinimalTransferAmountFromVested(amount, from, a.Type().ID)
		}

		e.log.Debug("cannot transfer funds, less than minimal amount requested to transfer",
			logging.BigUint("min-amount", minAmount),
			logging.BigUint("requested-amount", amount),
		)
		return fmt.Errorf("could not transfer funds, less than minimal amount requested to transfer")
	}

	return nil
}

func (e *Engine) ensureMinimalTransferAmountFromVested(
	transferAmount *num.Uint,
	from, asset string,
) error {
	account, err := e.col.GetPartyVestedRewardAccount(from, asset)
	if err != nil {
		return err
	}

	if transferAmount.EQ(account.Balance) {
		return nil
	}

	return fmt.Errorf("transfer from vested account under minimal transfer amount must be the full balance")
}

func (e *Engine) processTransfer(
	ctx context.Context,
	asset *assets.Asset,
	from, to, toMarket string,
	fromAcc, toAcc types.AccountType,
	amount *num.Uint,
	reference string,
	transferID string,
	epoch uint64,
	// optional from derived key transfer
	fromDerivedKey *string,
	// optional oneoff transfer
	// in case we need to schedule the delivery
	oneoff *types.OneOffTransfer,
) ([]*types.LedgerMovement, error) {
	assetType := asset.ToAssetType()

	// ensure the party have enough funds for both the
	// amount and the fee for the transfer
	feeTransfer, discount, err := e.makeFeeTransferForFundsTransfer(ctx, assetType, amount, from, fromAcc, fromDerivedKey, to)
	if err != nil {
		return nil, fmt.Errorf("could not pay the fee for transfer: %w", err)
	}
	feeTransferAccountType := []types.AccountType{fromAcc}

	// transfer from sub account to owners general account
	if fromDerivedKey != nil {
		from = *fromDerivedKey
	}

	fromTransfer, toTransfer := e.makeTransfers(from, to, assetType.ID, "", toMarket, amount, &transferID)
	transfers := []*types.Transfer{fromTransfer}
	accountTypes := []types.AccountType{fromAcc}
	references := []string{reference}

	// does the transfer needs to be finalized now?
	now := e.timeService.GetTimeNow()
	if oneoff == nil || (oneoff.DeliverOn == nil || oneoff.DeliverOn.Before(now)) {
		transfers = append(transfers, toTransfer)
		accountTypes = append(accountTypes, toAcc)
		references = append(references, reference)
		// if this goes well the whole transfer will be done
		// so we can set it to the proper status
	} else {
		// schedule the transfer
		e.scheduleTransfer(
			oneoff,
			toTransfer,
			toAcc,
			reference,
			*oneoff.DeliverOn,
		)
	}

	// process the transfer
	tresps, err := e.col.TransferFunds(
		ctx, transfers, accountTypes, references, []*types.Transfer{feeTransfer}, feeTransferAccountType,
	)
	if err != nil {
		return nil, err
	}

	e.broker.Send(events.NewTransferFeesEvent(ctx, transferID, feeTransfer.Amount.Amount, discount, epoch))

	return tresps, nil
}

func (e *Engine) makeTransfers(
	from, to, asset, fromMarket, toMarket string,
	amount *num.Uint,
	transferID *string,
) (*types.Transfer, *types.Transfer) {
	return &types.Transfer{
			Owner: from,
			Amount: &types.FinancialAmount{
				Amount: amount.Clone(),
				Asset:  asset,
			},
			Type:       types.TransferTypeTransferFundsSend,
			MinAmount:  amount.Clone(),
			Market:     fromMarket,
			TransferID: transferID,
		}, &types.Transfer{
			Owner: to,
			Amount: &types.FinancialAmount{
				Amount: amount.Clone(),
				Asset:  asset,
			},
			Type:       types.TransferTypeTransferFundsDistribute,
			MinAmount:  amount.Clone(),
			Market:     toMarket,
			TransferID: transferID,
		}
}

func (e *Engine) calculateFeeTransferForTransfer(
	asset *types.Asset,
	amount *num.Uint,
	from string,
	fromAccountType types.AccountType,
	fromDerivedKey *string,
	to string,
) *num.Uint {
	return calculateFeeForTransfer(
		asset.Details.Quantum,
		e.maxQuantumAmount,
		e.transferFeeFactor,
		amount,
		from,
		fromAccountType,
		fromDerivedKey,
		to,
	)
}

func (e *Engine) makeFeeTransferForFundsTransfer(
	ctx context.Context,
	asset *types.Asset,
	amount *num.Uint,
	from string,
	fromAccountType types.AccountType,
	fromDerivedKey *string,
	to string,
) (*types.Transfer, *num.Uint, error) {
	theoreticalFee := e.calculateFeeTransferForTransfer(asset, amount, from, fromAccountType, fromDerivedKey, to)
	feeAmount, discountAmount := e.ApplyFeeDiscount(ctx, asset.ID, from, theoreticalFee)

	if err := e.ensureEnoughFundsForTransfer(asset, amount, from, fromAccountType, fromDerivedKey, feeAmount); err != nil {
		return nil, nil, err
	}

	switch fromAccountType {
	case types.AccountTypeGeneral, types.AccountTypeVestedRewards, types.AccountTypeLockedForStaking:
	default:
		e.log.Panic("from account not supported",
			logging.String("account-type", fromAccountType.String()),
			logging.String("asset", asset.ID),
			logging.String("from", from),
		)
	}

	return &types.Transfer{
		Owner: from,
		Amount: &types.FinancialAmount{
			Amount: feeAmount.Clone(),
			Asset:  asset.ID,
		},
		Type:      types.TransferTypeInfrastructureFeePay,
		MinAmount: feeAmount,
	}, discountAmount, nil
}

func (e *Engine) ensureFeeForTransferFunds(
	asset *assets.Asset,
	amount *num.Uint,
	from string,
	fromAccountType types.AccountType,
	fromDerivedKey *string,
	to string,
) error {
	assetType := asset.ToAssetType()
	theoreticalFee := e.calculateFeeTransferForTransfer(assetType, amount, from, fromAccountType, fromDerivedKey, to)
	feeAmount, _ := e.EstimateFeeDiscount(assetType.ID, from, theoreticalFee)
	return e.ensureEnoughFundsForTransfer(assetType, amount, from, fromAccountType, fromDerivedKey, feeAmount)
}

func (e *Engine) ensureEnoughFundsForTransfer(
	asset *types.Asset,
	amount *num.Uint,
	from string,
	fromAccountType types.AccountType,
	fromDerivedKey *string,
	feeAmount *num.Uint,
) error {
	var (
		totalAmount = num.Sum(feeAmount, amount)
		account     *types.Account
		err         error
	)
	switch fromAccountType {
	case types.AccountTypeGeneral:
		account, err = e.col.GetPartyGeneralAccount(from, asset.ID)
		if err != nil {
			return err
		}
	case types.AccountTypeLockedForStaking:
		account, err = e.col.GetPartyLockedForStaking(from, asset.ID)
		if err != nil {
			return err
		}
	case types.AccountTypeVestedRewards:
		// sending from sub account to owners general account
		if fromDerivedKey != nil {
			from = *fromDerivedKey
		}

		account, err = e.col.GetPartyVestedRewardAccount(from, asset.ID)
		if err != nil {
			return err
		}

	default:
		e.log.Panic("from account not supported",
			logging.String("account-type", fromAccountType.String()),
			logging.String("asset", asset.ID),
			logging.String("from", from),
		)
	}

	if account.Balance.LT(totalAmount) {
		e.log.Debug("not enough funds to transfer",
			logging.BigUint("amount", amount),
			logging.BigUint("fee", feeAmount),
			logging.BigUint("total-amount", totalAmount),
			logging.BigUint("account-balance", account.Balance),
			logging.String("account-type", fromAccountType.String()),
			logging.String("asset", asset.ID),
			logging.String("from", from),
		)
		return ErrNotEnoughFundsToTransfer
	}
	return nil
}
