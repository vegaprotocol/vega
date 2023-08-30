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
	"fmt"

	"code.vegaprotocol.io/vega/core/assets"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
)

func (e *Engine) OnTransferFeeFactorUpdate(ctx context.Context, f num.Decimal) error {
	e.transferFeeFactor = f
	return nil
}

func (e *Engine) OnMinTransferQuantumMultiple(ctx context.Context, f num.Decimal) error {
	e.minTransferQuantumMultiple = f
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

	if err := e.ensureMinimalTransferAmount(a, t.Amount); err != nil {
		return err
	}

	_, err = e.ensureFeeForTransferFunds(t.Amount, t.From, t.Asset, t.FromAccountType)
	if err != nil {
		return fmt.Errorf("could not transfer funds, %w", err)
	}
	return nil
}

func (e *Engine) ensureMinimalTransferAmount(a *assets.Asset, amount *num.Uint) error {
	quantum := a.Type().Details.Quantum
	// no reason this would produce an error
	minAmount, _ := num.UintFromDecimal(quantum.Mul(e.minTransferQuantumMultiple))

	// no verify amount
	if amount.LT(minAmount) {
		e.log.Debug("cannot transfer funds, less than minimal amount requested to transfer",
			logging.BigUint("min-amount", minAmount),
			logging.BigUint("requested-amount", amount),
		)
		return fmt.Errorf("could not transfer funds, less than minimal amount requested to transfer")
	}

	return nil
}

func (e *Engine) processTransfer(
	ctx context.Context,
	from, to, asset, toMarket string,
	fromAcc, toAcc types.AccountType,
	amount *num.Uint,
	reference string,
	// optional oneoff transfer
	// in case we need to schedule the delivery
	oneoff *types.OneOffTransfer,
) ([]*types.LedgerMovement, error) {
	// ensure the party have enough funds for both the
	// amount and the fee for the transfer
	feeTransfer, err := e.ensureFeeForTransferFunds(amount, from, asset, fromAcc)
	if err != nil {
		return nil, fmt.Errorf("could not pay the fee for transfer: %w", err)
	}
	feeTransferAccountType := []types.AccountType{fromAcc}

	fromTransfer, toTransfer := e.makeTransfers(from, to, asset, "", toMarket, amount)
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

	return tresps, nil
}

func (e *Engine) makeTransfers(
	from, to, asset, fromMarket, toMarket string,
	amount *num.Uint,
) (*types.Transfer, *types.Transfer) {
	return &types.Transfer{
			Owner: from,
			Amount: &types.FinancialAmount{
				Amount: amount.Clone(),
				Asset:  asset,
			},
			Type:      types.TransferTypeTransferFundsSend,
			MinAmount: amount.Clone(),
			Market:    fromMarket,
		}, &types.Transfer{
			Owner: to,
			Amount: &types.FinancialAmount{
				Amount: amount.Clone(),
				Asset:  asset,
			},
			Type:      types.TransferTypeTransferFundsDistribute,
			MinAmount: amount.Clone(),
			Market:    toMarket,
		}
}

func (e *Engine) makeFeeTransferForTransferFunds(
	amount *num.Uint,
	from, asset string,
	fromAccountType types.AccountType,
) *types.Transfer {
	// first we calculate the fee
	feeAmount, _ := num.UintFromDecimal(amount.ToDecimal().Mul(e.transferFeeFactor))

	switch fromAccountType {
	case types.AccountTypeGeneral:
	default:
		e.log.Panic("from account not supported",
			logging.String("account-type", fromAccountType.String()),
			logging.String("asset", asset),
			logging.String("from", from),
		)
	}

	return &types.Transfer{
		Owner: from,
		Amount: &types.FinancialAmount{
			Amount: feeAmount.Clone(),
			Asset:  asset,
		},
		Type:      types.TransferTypeInfrastructureFeePay,
		MinAmount: feeAmount,
	}
}

func (e *Engine) ensureFeeForTransferFunds(
	amount *num.Uint,
	from, asset string,
	fromAccountType types.AccountType,
) (*types.Transfer, error) {
	transfer := e.makeFeeTransferForTransferFunds(
		amount, from, asset, fromAccountType,
	)

	var (
		totalAmount = num.Sum(transfer.Amount.Amount, amount)
		account     *types.Account
		err         error
	)
	switch fromAccountType {
	case types.AccountTypeGeneral:
		account, err = e.col.GetPartyGeneralAccount(from, asset)
		if err != nil {
			return nil, err
		}

	default:
		e.log.Panic("from account not supported",
			logging.String("account-type", fromAccountType.String()),
			logging.String("asset", asset),
			logging.String("from", from),
		)
	}

	if account.Balance.LT(totalAmount) {
		e.log.Debug("not enough funds to transfer",
			logging.BigUint("amount", amount),
			logging.BigUint("fee", transfer.Amount.Amount),
			logging.BigUint("total-amount", totalAmount),
			logging.BigUint("account-balance", account.Balance),
			logging.String("account-type", fromAccountType.String()),
			logging.String("asset", asset),
			logging.String("from", from),
		)
		return nil, ErrNotEnoughFundsToTransfer
	}

	return transfer, nil
}
