package events

import (
	"context"

	eventspb "code.vegaprotocol.io/protos/vega/events/v1"

	"code.vegaprotocol.io/vega/types"
)

type ERC20MultiSigSigner struct {
	*Base
	evt eventspb.ERC20MultiSigSignerEvent
}

func NewERC20MultiSigSigner(ctx context.Context, evt types.SignerEvent) *ERC20MultiSigSigner {
	return &ERC20MultiSigSigner{
		Base: newBase(ctx, ERC20MultiSigSignerEvent),
		evt:  *(evt.IntoProto()),
	}
}

func (s ERC20MultiSigSigner) ERC20MultiSigSigner() eventspb.ERC20MultiSigSignerEvent {
	return s.evt
}

func (s ERC20MultiSigSigner) Proto() eventspb.ERC20MultiSigSignerEvent {
	return s.evt
}

func (s ERC20MultiSigSigner) StreamMessage() *eventspb.BusEvent {
	busEvent := newBusEventFromBase(s.Base)
	busEvent.Event = &eventspb.BusEvent_Erc20MultisigSignerEvent{
		Erc20MultisigSignerEvent: &s.evt,
	}

	return busEvent
}

func ERC20MultiSigSignerFromStream(ctx context.Context, be *eventspb.BusEvent) *ERC20MultiSigSigner {
	return &ERC20MultiSigSigner{
		Base: newBaseFromBusEvent(ctx, ERC20MultiSigSignerEvent, be),
		evt:  *be.GetErc20MultisigSignerEvent(),
	}
}
