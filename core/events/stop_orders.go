package events

import (
	"context"

	"code.vegaprotocol.io/vega/core/types"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

type StopOrder struct {
	*Base
	so *eventspb.StopOrderEvent
}

func NewStopOrderEvent(ctx context.Context, so *types.StopOrder) *StopOrder {
	stop := &StopOrder{
		Base: newBase(ctx, StopOrderEvent),
		so:   so.ToProtoEvent(),
	}

	return stop
}

func (o StopOrder) StopOrder() *eventspb.StopOrderEvent {
	return o.so
}

func (o StopOrder) StreamMessage() *eventspb.BusEvent {
	busEvent := newBusEventFromBase(o.Base)
	busEvent.Event = &eventspb.BusEvent_StopOrder{
		StopOrder: o.so,
	}
	return busEvent
}

func StopOrderEventFromStream(ctx context.Context, be *eventspb.BusEvent) *StopOrder {
	stop := &StopOrder{
		Base: newBaseFromBusEvent(ctx, StopOrderEvent, be),
		so:   be.GetStopOrder(),
	}
	return stop
}
