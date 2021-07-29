package events

import (
	"context"

	ptypes "code.vegaprotocol.io/protos/vega"
	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	"code.vegaprotocol.io/data-node/types"
)

type Order struct {
	*Base
	o *ptypes.Order
}

func NewOrderEvent(ctx context.Context, o *types.Order) *Order {
	order := &Order{
		Base: newBase(ctx, OrderEvent),
		o:    o.IntoProto(),
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

func (o *Order) Order() *ptypes.Order {
	return o.o
}

func (o Order) Proto() ptypes.Order {
	return *o.o
}

func (o Order) StreamMessage() *eventspb.BusEvent {
	return &eventspb.BusEvent{
		Id:    o.eventID(),
		Block: o.TraceID(),
		Type:  o.et.ToProto(),
		Event: &eventspb.BusEvent_Order{
			Order: o.o,
		},
	}
}
