package events

import (
	"context"

	ptypes "code.vegaprotocol.io/vega/proto"
	eventspb "code.vegaprotocol.io/vega/proto/events/v1"
	"code.vegaprotocol.io/vega/types"
)

type Trade struct {
	*Base
	t ptypes.Trade
}

func NewTradeEvent(ctx context.Context, t types.Trade) *Trade {
	cpy := t.DeepClone()
	return &Trade{
		Base: newBase(ctx, TradeEvent),
		t:    *(t.IntoProto()),
	}
}

func (t Trade) MarketID() string {
	return t.t.MarketId
}

func (t Trade) IsParty(id string) bool {
	return t.t.Buyer == id || t.t.Seller == id
}

func (t *Trade) Trade() ptypes.Trade {
	return t.t
}

func (t Trade) Proto() ptypes.Trade {
	return t.t
}

func (t Trade) StreamMessage() *eventspb.BusEvent {
	return &eventspb.BusEvent{
		Id:    t.eventID(),
		Block: t.TraceID(),
		Type:  t.et.ToProto(),
		Event: &eventspb.BusEvent_Trade{
			Trade: &t.t,
		},
	}
}
