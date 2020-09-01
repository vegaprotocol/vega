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

func (o *Order) Order() *types.Order {
	return o.o
}

func (o Order) Proto() types.Order {
	return *o.o
}
