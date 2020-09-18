package events

import (
	"context"
	"fmt"

	types "code.vegaprotocol.io/vega/proto"
)

type Market struct {
	*Base
	m types.Market
}

func NewMarketEvent(ctx context.Context, m types.Market) *Market {
	return &Market{
		Base: newBase(ctx, MarketCreatedEvent),
		m:    m,
	}
}

// MarketEvent -> is needs to be logged as a market event
func (m Market) MarketEvent() string {
	return fmt.Sprintf("Market ID %s created (%s)", m.m.Id, m.m.String())
}

func (m Market) MarketID() string {
	return m.m.Id
}

func (m Market) Market() types.Market {
	return m.m
}

func (m Market) Proto() types.Market {
	return m.m
}

func (m Market) MarketProto() types.MarketEvent {
	return types.MarketEvent{
		MarketID: m.m.Id,
		Payload:  m.MarketEvent(),
	}
}

func (m Market) StreamMessage() *types.BusEvent {
	p := m.MarketProto()
	return &types.BusEvent{
		ID:   m.traceID,
		Type: m.et.ToProto(),
		Event: &types.BusEvent_Market{
			Market: &p,
		},
	}
}
