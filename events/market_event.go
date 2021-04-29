package events

import (
	"context"
	"fmt"

	types "code.vegaprotocol.io/vega/proto"
	eventspb "code.vegaprotocol.io/vega/proto/events/v1"
)

type MarketCreated struct {
	*Base
	m types.Market
}

func NewMarketCreatedEvent(ctx context.Context, m types.Market) *MarketCreated {
	cpy := m.DeepClone()
	return &MarketCreated{
		Base: newBase(ctx, MarketCreatedEvent),
		m:    *cpy,
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

func (m MarketCreated) MarketProto() eventspb.MarketEvent {
	return eventspb.MarketEvent{
		MarketId: m.m.Id,
		Payload:  m.MarketEvent(),
	}
}

func (m MarketCreated) StreamMessage() *eventspb.BusEvent {
	p := m.MarketProto()
	return &eventspb.BusEvent{
		Id:    m.eventID(),
		Block: m.TraceID(),
		Type:  m.et.ToProto(),
		Event: &eventspb.BusEvent_Market{
			Market: &p,
		},
	}
}

func (m MarketCreated) StreamMarketMessage() *eventspb.BusEvent {
	return m.StreamMessage()
}
