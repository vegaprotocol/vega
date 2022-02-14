package events

import (
	"context"

	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	"code.vegaprotocol.io/vega/types"
)

type EpochEvent struct {
	*Base
	e *eventspb.EpochEvent
}

func NewEpochEvent(ctx context.Context, e *types.Epoch) *EpochEvent {
	epoch := &EpochEvent{
		Base: newBase(ctx, EpochUpdate),
		e:    e.IntoProto(),
	}
	return epoch
}

func (e *EpochEvent) Epoch() *eventspb.EpochEvent {
	return e.e
}

func (e EpochEvent) Proto() eventspb.EpochEvent {
	return *e.e
}

func (e EpochEvent) StreamMessage() *eventspb.BusEvent {
	busEvent := newBusEventFromBase(e.Base)
	busEvent.Event = &eventspb.BusEvent_EpochEvent{
		EpochEvent: e.e,
	}
	return busEvent
}

func EpochEventFromStream(ctx context.Context, be *eventspb.BusEvent) *EpochEvent {
	return &EpochEvent{
		Base: newBaseFromBusEvent(ctx, EpochUpdate, be),
		e:    be.GetEpochEvent(),
	}
}
