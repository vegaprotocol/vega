package events

import (
	"context"

	eventspb "code.vegaprotocol.io/protos/vega/events/v1"

	"code.vegaprotocol.io/vega/types"
)

type ERC20MultiSigThresholdSet struct {
	*Base
	evt eventspb.ERC20MultiSigThresholdSetEvent
}

func NewERC20MultiSigThresholdSet(ctx context.Context, evt types.SignerThresholdSetEvent) *ERC20MultiSigThresholdSet {
	return &ERC20MultiSigThresholdSet{
		Base: newBase(ctx, ERC20MultiSigThresholdSetEvent),
		evt:  *(evt.IntoProto()),
	}
}

func (s ERC20MultiSigThresholdSet) ERC20MultiSigThresholdSet() eventspb.ERC20MultiSigThresholdSetEvent {
	return s.evt
}

func (s ERC20MultiSigThresholdSet) Proto() eventspb.ERC20MultiSigThresholdSetEvent {
	return s.evt
}

func (s ERC20MultiSigThresholdSet) StreamMessage() *eventspb.BusEvent {
	busEvent := newBusEventFromBase(s.Base)
	busEvent.Event = &eventspb.BusEvent_Erc20MultisigSetThresholdEvent{
		Erc20MultisigSetThresholdEvent: &s.evt,
	}

	return busEvent
}

func ERC20MultiSigThresholdSetFromStream(ctx context.Context, be *eventspb.BusEvent) *ERC20MultiSigThresholdSet {
	return &ERC20MultiSigThresholdSet{
		Base: newBaseFromBusEvent(ctx, ERC20MultiSigThresholdSetEvent, be),
		evt:  *be.GetErc20MultisigSetThresholdEvent(),
	}
}
