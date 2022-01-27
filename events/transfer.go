package events

import (
	"context"

	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	"code.vegaprotocol.io/vega/types"
)

// Transfer ...
type TransferFunds struct {
	*Base
	transfer *eventspb.Transfer
}

func NewOneOffTransferFundsEvent(
	ctx context.Context,
	t *types.OneOffTransfer,
) *TransferFunds {
	return &TransferFunds{
		Base:     newBase(ctx, TransferEvent),
		transfer: t.IntoEvent(),
	}
}

func NewRecurringTransferFundsEvent(
	ctx context.Context,
	t *types.RecurringTransfer,
) *TransferFunds {
	return &TransferFunds{
		Base:     newBase(ctx, TransferEvent),
		transfer: t.IntoEvent(),
	}
}

func (t TransferFunds) PartyID() string {
	return t.transfer.From
}

func (t TransferFunds) TransferFunds() eventspb.Transfer {
	return t.Proto()
}

func (t TransferFunds) Proto() eventspb.Transfer {
	return *t.transfer
}

func (t TransferFunds) StreamMessage() *eventspb.BusEvent {
	p := t.Proto()

	busEvent := newBusEventFromBase(t.Base)
	busEvent.Event = &eventspb.BusEvent_Transfer{
		Transfer: &p,
	}

	return busEvent
}

func TransferFundsEventFromStream(ctx context.Context, be *eventspb.BusEvent) *TransferFunds {
	event := be.GetTransfer()
	if event == nil {
		return nil
	}

	return &TransferFunds{
		Base:     newBaseFromBusEvent(ctx, TransferEvent, be),
		transfer: event,
	}
}
