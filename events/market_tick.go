package events

import (
	"context"
	"fmt"
	"time"

	eventspb "code.vegaprotocol.io/vega/proto/events/v1"
)

type MarketTick struct {
	*Base
	id string
	t  time.Time
}

func NewMarketTick(ctx context.Context, id string, t time.Time) *MarketTick {
	return &MarketTick{
		Base: newBase(ctx, MarketTickEvent),
		id:   id,
		t:    t,
	}
}

func (m MarketTick) MarketID() string {
	return m.id
}

func (m MarketTick) Time() time.Time {
	return m.t
}

func (m MarketTick) MarketEvent() string {
	return fmt.Sprintf("Market %s on time %s", m.id, m.t.String())
}

func (m MarketTick) Proto() eventspb.MarketTick {
	return eventspb.MarketTick{
		Id:   m.id,
		Time: m.t.UTC().Unix(),
	}
}

func (m MarketTick) MarketProto() eventspb.MarketEvent {
	return eventspb.MarketEvent{
		MarketId: m.id,
		Payload:  m.MarketEvent(),
	}
}

func (m MarketTick) StreamMessage() *eventspb.BusEvent {
	p := m.Proto()
	return &eventspb.BusEvent{
		Id:    m.eventID(),
		Block: m.TraceID(),
		Type:  m.et.ToProto(),
		Event: &eventspb.BusEvent_MarketTick{
			MarketTick: &p,
		},
	}
}

func (m MarketTick) StreamMarketMessage() *eventspb.BusEvent {
	return m.StreamMessage()
}
