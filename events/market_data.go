package events

import (
	"context"

	proto "code.vegaprotocol.io/protos/vega"
	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	"code.vegaprotocol.io/vega/types"
)

type MarketData struct {
	*Base
	md proto.MarketData
}

func NewMarketDataEvent(ctx context.Context, md types.MarketData) *MarketData {
	return &MarketData{
		Base: newBase(ctx, MarketDataEvent),
		md:   *md.IntoProto(),
	}
}

func (m MarketData) MarketID() string {
	return m.md.Market
}

func (m MarketData) MarketData() proto.MarketData {
	return m.md
}

func (m MarketData) Proto() proto.MarketData {
	return m.md
}

func (m MarketData) StreamMessage() *eventspb.BusEvent {
	return &eventspb.BusEvent{
		Version: eventspb.Version,
		Id:      m.eventID(),
		Block:   m.TraceID(),
		ChainId: m.ChainID(),
		Type:    m.et.ToProto(),
		Event: &eventspb.BusEvent_MarketData{
			MarketData: &m.md,
		},
	}
}

func MarketDataEventFromStream(ctx context.Context, be *eventspb.BusEvent) *MarketData {
	return &MarketData{
		Base: newBaseFromStream(ctx, MarketDataEvent, be),
		md:   *be.GetMarketData(),
	}
}
