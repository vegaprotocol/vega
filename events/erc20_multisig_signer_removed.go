package events

import (
	"context"

	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
)

type ERC20MultiSigSignerRemoved struct {
	*Base
	evt eventspb.ERC20MultiSigSignerRemoved
}

func NewERC20MultiSigSignerRemoved(ctx context.Context, evt eventspb.ERC20MultiSigSignerRemoved) *ERC20MultiSigSignerRemoved {
	return &ERC20MultiSigSignerRemoved{
		Base: newBase(ctx, ERC20MultiSigSignerRemovedEvent),
		evt:  evt,
	}
}

func (s ERC20MultiSigSignerRemoved) ERC20MultiSigSignerRemoved() eventspb.ERC20MultiSigSignerRemoved {
	return s.evt
}

func (s ERC20MultiSigSignerRemoved) Proto() eventspb.ERC20MultiSigSignerRemoved {
	return s.evt
}

func (s ERC20MultiSigSignerRemoved) StreamMessage() *eventspb.BusEvent {
	busEvent := newBusEventFromBase(s.Base)
	busEvent.Event = &eventspb.BusEvent_Erc20MultisigSignerRemoved{
		Erc20MultisigSignerRemoved: &s.evt,
	}
	return busEvent
}

func ERC20MultiSigSignerRemovedFromStream(ctx context.Context, be *eventspb.BusEvent) *ERC20MultiSigSignerRemoved {
	return &ERC20MultiSigSignerRemoved{
		Base: newBaseFromBusEvent(ctx, ERC20MultiSigSignerRemovedEvent, be),
		evt:  *be.GetErc20MultisigSignerRemoved(),
	}
}
