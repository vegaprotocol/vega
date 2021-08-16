package events

import (
	"context"
	"fmt"

	proto "code.vegaprotocol.io/protos/vega"
	eventspb "code.vegaprotocol.io/protos/vega/events/v1"

	"code.vegaprotocol.io/vega/types"
)

type MarketCreated struct {
	*Base
	m  types.Market
	pm proto.Market
}

func NewMarketCreatedEvent(ctx context.Context, m types.Market) *MarketCreated {
	pm := m.IntoProto()
	return &MarketCreated{
		Base: newBase(ctx, MarketCreatedEvent),
		m:    m,
		pm:   *pm,
	}
}

// MarketEvent -> is needs to be logged as a market event
func (m MarketCreated) MarketEvent() string {
	return fmt.Sprintf("Market ID %s created (%s)", m.m.ID, m.pm.String())
}

func (m MarketCreated) MarketID() string {
	return m.m.ID
}

func (m MarketCreated) Market() proto.Market {
	return m.pm
}

func (m MarketCreated) Proto() proto.Market {
	return m.pm
}

func (m MarketCreated) MarketProto() eventspb.MarketEvent {
	return eventspb.MarketEvent{
		MarketId: m.m.ID,
		Payload:  m.MarketEvent(),
	}
}

func (m MarketCreated) StreamMessage() *eventspb.BusEvent {
	market := m.Proto()
	return &eventspb.BusEvent{
		Id:    m.eventID(),
		Block: m.TraceID(),
		Type:  m.et.ToProto(),
		Event: &eventspb.BusEvent_MarketCreated{
			MarketCreated: &market,
		},
	}
}

func (m MarketCreated) StreamMarketMessage() *eventspb.BusEvent {
	return m.StreamMessage()
}

func MarketCreatedEventFromStream(ctx context.Context, be *eventspb.BusEvent) *MarketCreated {
	m := be.GetMarketCreated()
	return &MarketCreated{
		Base: newBaseFromStream(ctx, MarketCreatedEvent, be),
		m:    types.Market{ID: m.Id},
		pm:   *m,
	}
}
