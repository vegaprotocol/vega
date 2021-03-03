package events

import (
	"context"

	types "code.vegaprotocol.io/vega/proto"
)

type Order struct {
	*Base
	o types.Order
}

func NewOrderEvent(ctx context.Context, o *types.Order) *Order {
	order := &Order{
		Base: newBase(ctx, OrderEvent),
		o:    *o,
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

func (o Order) StreamMessage() *types.BusEvent {
	cpy := o.o
	return &types.BusEvent{
		Id:    o.eventID(),
		Block: o.TraceID(),
		Type:  o.et.ToProto(),
		Event: &types.BusEvent_Order{
			Order: &cpy,
		},
	}
}
