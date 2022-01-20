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

var (
	ErrStartEpochInThePast = errors.New("start epoch in the past")
)

type transferForEpoch struct {
	// transfers using the same account type
	transfer              *types.Transfer
	accountType           types.AccountType
	feeTransfer           *types.Transfer
	feeTransferAccounType types.AccountType
}

type recurringTransfer struct {
	// to send events
	recurring *types.RecurringTransfer
	// epoch to transfers
	transfers map[uint64]*transferForEpoch
	reference string
}

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

	// now calculate the transfers
	var (
		tfe         = map[uint64]*transferForEpoch{}
		startAmount = transfer.Amount.ToDecimal()
		startEpoch  = num.NewUint(transfer.StartEpoch).ToDecimal()
		payTransfer *types.Transfer
		totalFee    = num.Zero()
	)

	for i := transfer.StartEpoch; transfer.EndEpoch > i; i++ {
		currentEpoch := num.NewUint(i).ToDecimal()
		amount, _ := num.UintFromDecimal(startAmount.Mul(transfer.Factor.Pow(currentEpoch.Sub(startEpoch))))

		fromTransfer, toTransfer := e.makeTransfers(
			transfer.From, transfer.To, transfer.Asset, amount)

		if payTransfer != nil {
			payTransfer = payTransfer.Merge(fromTransfer)
		} else {
			payTransfer = fromTransfer

		}

		feeTransfer := e.makeFeeTransferForTransferFunds(
			amount.Clone(), transfer.From, transfer.Asset, transfer.FromAccountType,
		)

		// add the new fee to the total amount
		totalFee.Add(totalFee, feeTransfer.Amount.Amount)

		// add the map
		tfe[i] = transferForEpoch{
			transfer:              toTransfer,
			accountType:           transfer.ToAccountType,
			feeTransfer:           feeTransfer,
			feeTransferAccounType: transfer.FromAccountType,
		}

	}

	return errors.New("unimplemented")
}

func (e *Engine) distributeRecurringTransfers(
	ctx context.Context,
	newEpoch uint64,
) error {
	var (
		evts         = []events.Event{}
		transfers    = []*types.Transfer{}
		feeTransfers = []*types.Transfer{}
		accountTypes = []types.AccountType{}
		references   = []string{}
	)

	// iterate over all transfers
	for k, v := range e.recurringTransfers {
		tfe, ok := v.transfers[newEpoch]
		if !ok {
			// no transfer for this epoch
			continue
		}
		reference := v.reference

		// delete the transfer from the map
		delete(v.transfers, newEpoch)

		// if we don't have anymore
		if len(v.transfers) <= 0 {
			v.recurring.Status = types.TransferStatusDone
			evts = append(evts, events.NewRecurringTransferFundsEvent(ctx, v.recurring))
			delete(e.recurringTransfers, k)
		}

		// add to slices
		transfers = append(transfers, tfe.transfer)
		feeTransfers = append(feeTransfers, tfe.feeTransfer)
		accountTypes = append(accountTypes, tfe.accountType)
		references = append(references, reference)
	}

	// call collateral
	tresps, err := e.col.TransferFunds(
		ctx, transfers, accountTypes, references, feeTransfers, accountTypes,
	)
	if err != nil {
		return err
	}

	// send events
	e.broker.Send(events.NewTransferResponse(ctx, tresps))
	e.broker.SendBatch(evts)

	return nil
}
