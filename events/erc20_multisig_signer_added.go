package events

import (
	"context"

	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
)

type ERC20MultiSigSignerAdded struct {
	*Base
	evt eventspb.ERC20MultiSigSignerAdded
}

func NewERC20MultiSigSignerAdded(ctx context.Context, evt eventspb.ERC20MultiSigSignerAdded) *ERC20MultiSigSignerAdded {
	return &ERC20MultiSigSignerAdded{
		Base: newBase(ctx, ERC20MultiSigSignerAddedEvent),
		evt:  evt,
	}
}

func (s ERC20MultiSigSignerAdded) ERC20MultiSigSignerAdded() eventspb.ERC20MultiSigSignerAdded {
	return s.evt
}

func (s ERC20MultiSigSignerAdded) Proto() eventspb.ERC20MultiSigSignerAdded {
	return s.evt
}

func (s ERC20MultiSigSignerAdded) StreamMessage() *eventspb.BusEvent {
	busEvent := newBusEventFromBase(s.Base)
	busEvent.Event = &eventspb.BusEvent_Erc20MultisigSignerAdded{
		Erc20MultisigSignerAdded: &s.evt,
	}
	return busEvent
}

func ERC20MultiSigSignerAddedFromStream(ctx context.Context, be *eventspb.BusEvent) *ERC20MultiSigSignerAdded {
	return &ERC20MultiSigSignerAdded{
		Base: newBaseFromBusEvent(ctx, ERC20MultiSigSignerAddedEvent, be),
		evt:  *be.GetErc20MultisigSignerAdded(),
	}
}
