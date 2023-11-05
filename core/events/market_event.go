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

	"code.vegaprotocol.io/vega/core/types"
	proto "code.vegaprotocol.io/vega/protos/vega"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
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

// MarketEvent -> is needs to be logged as a market event.
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

	busEvent := newBusEventFromBase(m.Base)
	busEvent.Event = &eventspb.BusEvent_MarketCreated{
		MarketCreated: &market,
	}

	return busEvent
}

func (m MarketCreated) StreamMarketMessage() *eventspb.BusEvent {
	return m.StreamMessage()
}

func MarketCreatedEventFromStream(ctx context.Context, be *eventspb.BusEvent) *MarketCreated {
	m := be.GetMarketCreated()
	return &MarketCreated{
		Base: newBaseFromBusEvent(ctx, MarketCreatedEvent, be),
		m:    types.Market{ID: m.Id},
		pm:   *m,
	}
}
