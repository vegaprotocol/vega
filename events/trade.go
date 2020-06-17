package events

import (
	"context"

	types "code.vegaprotocol.io/vega/proto"
)

type Trade struct {
	*Base
	t types.Trade
}

func NewTradeEvent(ctx context.Context, t types.Trade) *Trade {
	return &Order{
		Base: newBase(ctx, OrderEvent),
		t:    t,
	}
}

func (o *Trade) Trade() types.Trade {
	return t.t
}
