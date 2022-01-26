package banking

import (
	"context"
	"errors"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/types"
)

var (
	ErrRecurringTransferDoesNotExists             = errors.New("recurring transfer does not exists")
	ErrCannotCancelOtherPartiesRecurringTransfers = errors.New("cannot cancel other parties recurring transfers")
)

func (e *Engine) CancelTransferFunds(
	ctx context.Context,
	cancel *types.CancelTransferFunds,
) error {
	// validation is simple, does the transfer
	// exists
	transfer, ok := e.recurringTransfers[cancel.TransferID]
	if !ok {
		return ErrRecurringTransferDoesNotExists
	}

	// Is the From party of the transfer
	// the party which submitted the transaction?
	if transfer.From != cancel.Party {
		return ErrCannotCancelOtherPartiesRecurringTransfers
	}

	// all good, let's delete
	delete(e.recurringTransfers, cancel.TransferID)

	// send an event because we are nice with the data-node
	transfer.Status = types.TransferStatusCancelled
	e.broker.Send(events.NewRecurringTransferFundsEvent(ctx, transfer))

	return nil
}
