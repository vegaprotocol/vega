package events

import (
	"context"

	types "code.vegaprotocol.io/vega/proto"
)

type Order struct {
	*Base
	o *types.Order
}

func NewOrderEvent(ctx context.Context, o *types.Order) *Order {
	return &Order{
		Base: newBase(ctx, OrderEvent),
		o:    o,
	}
}

func (o Order) IsParty(id string) bool {
	return (o.o.PartyID == id)
}

func (o Order) PartyID() string {
	return o.o.PartyID
}

func (o Order) MarketID() string {
	return o.o.MarketID
}

func (o *Order) Order() *types.Order {
	return o.o
}

func (o Order) Proto() types.Order {
	return *o.o
}

func (o Order) StreamMessage() *types.BusEvent {
	cpy := *o.o
	return &types.BusEvent{
		ID:   o.eventID(),
		Type: o.et.ToProto(),
		Event: &types.BusEvent_Order{
			Order: &cpy,
		},
	}
}
