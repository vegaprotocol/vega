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

	"code.vegaprotocol.io/vega/core/assets"
	"code.vegaprotocol.io/vega/core/events"
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

	if err := e.ensureMinimalTransferAmount(a, t.Amount, t.FromAccountType, t.From); err != nil {
		return err
	}

	_, err = e.ensureFeeForTransferFunds(t.Amount, t.From, t.Asset, t.FromAccountType, t.To)
	if err != nil {
		return fmt.Errorf("could not transfer funds, %w", err)
	}
	return nil
}

func (e *Engine) ensureMinimalTransferAmount(
	a *assets.Asset,
	amount *num.Uint,
	fromAccType types.AccountType,
	from string,
) error {
	quantum := a.Type().Details.Quantum
	// no reason this would produce an error
	minAmount, _ := num.UintFromDecimal(quantum.Mul(e.minTransferQuantumMultiple))

	// no verify amount
	if amount.LT(minAmount) {
		if fromAccType == types.AccountTypeVestedRewards {
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
	from, to, asset, toMarket string,
	fromAcc, toAcc types.AccountType,
	amount *num.Uint,
	reference string,
	transferID string,
	epoch uint64,
	// optional oneoff transfer
	// in case we need to schedule the delivery
	oneoff *types.OneOffTransfer,
) ([]*types.LedgerMovement, error) {
	// ensure the party have enough funds for both the
	// amount and the fee for the transfer
	feeTransfer, err := e.ensureFeeForTransferFunds(amount, from, asset, fromAcc, to)
	if err != nil {
		return nil, fmt.Errorf("could not pay the fee for transfer: %w", err)
	}
	feeTransferAccountType := []types.AccountType{fromAcc}

	fromTransfer, toTransfer := e.makeTransfers(from, to, asset, "", toMarket, amount, &transferID)
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
	e.broker.Send(events.NewTransferFeesEvent(ctx, transferID, feeTransfer.Amount.Amount, epoch))

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

func (e *Engine) makeFeeTransferForTransferFunds(
	amount *num.Uint,
	from, asset string,
	fromAccountType types.AccountType,
	to string,
) *types.Transfer {
	// no fee for Vested account
	feeAmount := num.UintZero()

	// first we calculate the fee
	if !(fromAccountType == types.AccountTypeVestedRewards && from == to) {
		feeAmount, _ = num.UintFromDecimal(amount.ToDecimal().Mul(e.transferFeeFactor))
	}

	switch fromAccountType {
	case types.AccountTypeGeneral, types.AccountTypeVestedRewards:
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
	to string,
) (*types.Transfer, error) {
	transfer := e.makeFeeTransferForTransferFunds(
		amount, from, asset, fromAccountType, to,
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
	case types.AccountTypeVestedRewards:
		account, err = e.col.GetPartyVestedRewardAccount(from, asset)
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
