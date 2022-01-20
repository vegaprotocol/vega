package banking

import (
	"context"
	"errors"
	"fmt"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
)

var ErrStartEpochInThePast = errors.New("start epoch in the past")

func (e *Engine) recurringTransfer(
	ctx context.Context,
	transfer *types.RecurringTransfer,
) error {
	defer func() {
		e.broker.Send(events.NewRecurringTransferFundsEvent(ctx, transfer))
	}()

	// ensure asset exists
	if _, err := e.assets.Get(transfer.Asset); err != nil {
		transfer.Status = types.TransferStatusRejected
		e.log.Debug("cannot transfer funds, invalid asset", logging.Error(err))
		return fmt.Errorf("could not transfer funds: %w", err)
	}

	if err := transfer.IsValid(); err != nil {
		transfer.Status = types.TransferStatusRejected
		return err
	}

	// can't create transfer with start epoch in the past
	if transfer.StartEpoch <= e.currentEpoch {
		transfer.Status = types.TransferStatusRejected
		return ErrStartEpochInThePast
	}

	// from here all sounds OK, we can add the transfer
	// in the recurringTransfer map
	e.recurringTransfers[transfer.ID] = transfer

	return nil
}

func (e *Engine) makeRecurringFeeTransfer(
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

func (e *Engine) ensureRecurringTransferFee(
	amount, feeAmount *num.Uint,
	from, asset string,
	fromAccountType types.AccountType,
) error {
	var (
		totalAmount = num.Sum(amount, feeAmount)
		account     *types.Account
		err         error
	)
	switch fromAccountType {
	case types.AccountTypeGeneral:
		account, err = e.col.GetPartyGeneralAccount(from, asset)
		if err != nil {
			return err
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
			logging.BigUint("fee", feeAmount),
			logging.BigUint("total-amount", totalAmount),
			logging.BigUint("account-balance", account.Balance),
			logging.String("account-type", fromAccountType.String()),
			logging.String("asset", asset),
			logging.String("from", from),
		)
		return ErrNotEnoughFundsToTransfer
	}

	return nil
}

func (e *Engine) distributeRecurringTransfers(
	ctx context.Context,
	newEpoch uint64,
) error {
	var (
		transfersDone = []events.Event{}
		tresps        = []*types.TransferResponse{}
	)

	allIDs := make([]string, 0, len(e.recurringTransfers))
	for k := range e.recurringTransfers {
		allIDs = append(allIDs, k)
	}

	// iterate over all transfers
	for _, k := range allIDs {
		v := e.recurringTransfers[k]
		if v.StartEpoch < newEpoch {
			// not started
			continue
		}

		// call collateral
		resps, err := e.processTransfer(
			ctx, v.From, v.To, v.Asset, v.ToAccountType, v.ToAccountType, v.Amount.Clone(), v.Reference, nil, // last is eventual oneoff, which this is not
		)
		if err != nil {
			v.Status = types.TransferStatusStopped
			transfersDone = append(transfersDone,
				events.NewRecurringTransferFundsEvent(ctx, v))
			delete(e.recurringTransfers, k)
			continue
		}

		tresps = append(tresps, resps...)

		// if we don't have anymore
		if v.EndEpoch != nil && *v.EndEpoch == e.currentEpoch {
			v.Status = types.TransferStatusDone
			transfersDone = append(transfersDone, events.NewRecurringTransferFundsEvent(ctx, v))
			delete(e.recurringTransfers, k)
		}

	}

	// send events
	if len(tresps) > 0 {
		e.broker.Send(events.NewTransferResponse(ctx, tresps))
	}
	if len(transfersDone) > 0 {
		e.broker.SendBatch(transfersDone)
	}

	return nil
}
