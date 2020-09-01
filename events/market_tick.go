package events

import (
	"context"
	"fmt"
	"time"

	types "code.vegaprotocol.io/vega/proto"
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
