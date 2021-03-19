package events

import (
	"context"

	types "code.vegaprotocol.io/vega/proto"
	eventspb "code.vegaprotocol.io/vega/proto/events/v1"
)

type Order struct {
	*Base
	o types.Order
}

func NewOrderEvent(ctx context.Context, o *types.Order) *Order {
	cpy := o.DeepClone()
	order := &Order{
		Base: newBase(ctx, OrderEvent),
		o:    *cpy,
	}
	return order
}

func (o Order) IsParty(id string) bool {
	return o.o.PartyId == id
}

func (o Order) PartyID() string {
	return o.o.PartyId
}

func (o Order) MarketID() string {
	return o.o.MarketId
}

func (o *Order) Order() *types.Order {
	return &o.o
}

func (o Order) Proto() types.Order {
	return o.o
}

func (o Order) StreamMessage() *eventspb.BusEvent {
	cpy := o.o
	return &eventspb.BusEvent{
		Id:    o.eventID(),
		Block: o.TraceID(),
		Type:  o.et.ToProto(),
		Event: &eventspb.BusEvent_Order{
			Order: &cpy,
		},
	}
}
