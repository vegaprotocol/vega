// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package events

import (
	"context"
	"fmt"
	"time"

	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
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
