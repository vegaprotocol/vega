package events

import (
	"context"
	"fmt"

	proto "code.vegaprotocol.io/protos/vega"
	eventspb "code.vegaprotocol.io/protos/vega/events/v1"

	"code.vegaprotocol.io/vega/types"
)

type MarketUpdated struct {
	*Base
	m  types.Market
	pm proto.Market
}

func NewMarketUpdatedEvent(ctx context.Context, m types.Market) *MarketUpdated {
	pm := m.IntoProto()
	return &MarketUpdated{
		Base: newBase(ctx, MarketUpdatedEvent),
		m:    m,
		pm:   *pm,
	}
}

// MarketEvent -> is needs to be logged as a market event.
func (m MarketUpdated) MarketEvent() string {
	return fmt.Sprintf("Market ID %s updated (%s)", m.m.ID, m.pm.String())
}

func (m MarketUpdated) MarketID() string {
	return m.m.ID
}

func (m MarketUpdated) Market() proto.Market {
	return m.Proto()
}

func (m MarketUpdated) Proto() proto.Market {
	return m.pm
}

func (m MarketUpdated) MarketProto() eventspb.MarketEvent {
	return eventspb.MarketEvent{
		MarketId: m.m.ID,
		Payload:  m.MarketEvent(),
	}
}

func (m MarketUpdated) StreamMessage() *eventspb.BusEvent {
	market := m.Proto()
	return &eventspb.BusEvent{
		Version: eventspb.Version,
		Id:      m.eventID(),
		Block:   m.TraceID(),
		Type:    m.et.ToProto(),
		Event: &eventspb.BusEvent_MarketUpdated{
			MarketUpdated: &market,
		},
	}
}

func (m MarketUpdated) StreamMarketMessage() *eventspb.BusEvent {
	return m.StreamMessage()
}

func MarketUpdatedEventFromStream(ctx context.Context, be *eventspb.BusEvent) *MarketUpdated {
	m := be.GetMarketUpdated()
	return &MarketUpdated{
		Base: newBaseFromStream(ctx, MarketUpdatedEvent, be),
		m:    types.Market{ID: m.Id},
		pm:   *m,
	}
}
