// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package events

import (
	"context"
	"fmt"
	"time"

	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
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
	busEvent := newBusEventFromBase(m.Base)
	busEvent.Event = &eventspb.BusEvent_MarketTick{
		MarketTick: &p,
	}

	return busEvent
}

func (m MarketTick) StreamMarketMessage() *eventspb.BusEvent {
	return m.StreamMessage()
}

func MarketTickEventFromStream(ctx context.Context, be *eventspb.BusEvent) *MarketTick {
	return &MarketTick{
		Base: newBaseFromBusEvent(ctx, MarketTickEvent, be),
		id:   be.GetMarketTick().GetId(),
		t:    time.Unix(be.GetMarketTick().GetTime(), 0).UTC(),
	}
}
