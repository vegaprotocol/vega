package events

import (
	"context"
	"fmt"

	types "code.vegaprotocol.io/vega/proto"
)

type MarketCreated struct {
	*Base
	m types.Market
}

func NewMarketCreatedEvent(ctx context.Context, m types.Market) *MarketCreated {
	return &MarketCreated{
		Base: newBase(ctx, MarketCreatedEvent),
		m:    m,
	}
}

// MarketEvent -> is needs to be logged as a market event
func (m MarketCreated) MarketEvent() string {
	return fmt.Sprintf("Market ID %s created (%s)", m.m.Id, m.m.String())
}

func (m MarketCreated) MarketID() string {
	return m.m.Id
}

func (m MarketCreated) Market() types.Market {
	return m.m
}

func (m MarketCreated) Proto() types.Market {
	return m.m
}

func (m MarketCreated) MarketProto() types.MarketEvent {
	return types.MarketEvent{
		MarketId: m.m.Id,
		Payload:  m.MarketEvent(),
	}
}

func (m MarketCreated) StreamMessage() *types.BusEvent {
	p := m.MarketProto()
	return &types.BusEvent{
		Id:    m.eventID(),
		Block: m.TraceID(),
		Type:  m.et.ToProto(),
		Event: &types.BusEvent_Market{
			Market: &p,
		},
	}
}

func (m MarketCreated) StreamMarketMessage() *types.BusEvent {
	return m.StreamMessage()
}
