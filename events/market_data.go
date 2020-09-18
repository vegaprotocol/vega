package events

import (
	"context"

	types "code.vegaprotocol.io/vega/proto"
)

type MarketData struct {
	*Base
	md types.MarketData
}

func NewMarketDataEvent(ctx context.Context, md types.MarketData) *MarketData {
	return &MarketData{
		Base: newBase(ctx, MarketDataEvent),
		md:   md,
	}
}

func (m MarketData) MarketID() string {
	return m.md.Market
}

func (m MarketData) MarketData() types.MarketData {
	return m.md
}

func (m MarketData) Proto() types.MarketData {
	return m.md
}

func (m MarketData) StreamMessage() *types.BusEvent {
	return &types.BusEvent{
		ID:   m.traceID,
		Type: m.et.ToProto(),
		Event: &types.BusEvent_MarketData{
			MarketData: &m.md,
		},
	}
}
