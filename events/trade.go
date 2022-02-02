package events

import (
	"context"

	ptypes "code.vegaprotocol.io/protos/vega"
	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	"code.vegaprotocol.io/vega/types"
)

type Trade struct {
	*Base
	t ptypes.Trade
}

func NewTradeEvent(ctx context.Context, t types.Trade) *Trade {
	p := t.IntoProto()
	p.Price = t.MarketPrice.String()
	return &Trade{
		Base: newBase(ctx, TradeEvent),
		t:    *p,
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
	busEvent := newBusEventFromBase(t.Base)
	busEvent.Event = &eventspb.BusEvent_Trade{
		Trade: &t.t,
	}

	return busEvent
}

func TradeEventFromStream(ctx context.Context, be *eventspb.BusEvent) *Trade {
	return &Trade{
		Base: newBaseFromBusEvent(ctx, TradeEvent, be),
		t:    *be.GetTrade(),
	}
}
