package events

import (
	"context"
	"fmt"
	"time"

	types "code.vegaprotocol.io/vega/proto/gen/golang"
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

func (m MarketTick) Proto() types.MarketTick {
	return types.MarketTick{
		ID:   m.id,
		Time: m.t.UTC().Unix(),
	}
}

func (m MarketTick) MarketProto() types.MarketEvent {
	return types.MarketEvent{
		MarketID: m.id,
		Payload:  m.MarketEvent(),
	}
}

func (m MarketTick) StreamMessage() *types.BusEvent {
	p := m.Proto()
	return &types.BusEvent{
		ID:    m.eventID(),
		Block: m.TraceID(),
		Type:  m.et.ToProto(),
		Event: &types.BusEvent_MarketTick{
			MarketTick: &p,
		},
	}
}

func (m MarketTick) StreamMarketMessage() *types.BusEvent {
	return m.StreamMessage()
}
