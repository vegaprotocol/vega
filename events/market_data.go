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

func (m MarketData) MarketData() types.MarketData {
	return m.md
}
