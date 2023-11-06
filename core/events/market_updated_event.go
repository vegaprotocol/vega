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

type MarketUpdated struct {
	*Base
	pm proto.Market
}

func NewMarketUpdatedEvent(ctx context.Context, m types.Market) *MarketUpdated {
	pm := m.IntoProto()
	return &MarketUpdated{
		Base: newBase(ctx, MarketUpdatedEvent),
		pm:   *pm,
	}
}

// MarketEvent -> is needs to be logged as a market event.
func (m MarketUpdated) MarketEvent() string {
	return fmt.Sprintf("Market ID %s updated (%s)", m.pm.Id, m.pm.String())
}

func (m MarketUpdated) MarketID() string {
	return m.pm.Id
}

func (m MarketUpdated) Market() proto.Market {
	return m.Proto()
}

func (m MarketUpdated) Proto() proto.Market {
	return m.pm
}

func (m MarketUpdated) MarketProto() eventspb.MarketEvent {
	return eventspb.MarketEvent{
		MarketId: m.pm.Id,
		Payload:  m.MarketEvent(),
	}
}

func (m MarketUpdated) StreamMessage() *eventspb.BusEvent {
	market := m.Proto()
	busEvent := newBusEventFromBase(m.Base)
	busEvent.Event = &eventspb.BusEvent_MarketUpdated{
		MarketUpdated: &market,
	}

	return busEvent
}

func (m MarketUpdated) StreamMarketMessage() *eventspb.BusEvent {
	return m.StreamMessage()
}

func MarketUpdatedEventFromStream(ctx context.Context, be *eventspb.BusEvent) *MarketUpdated {
	m := be.GetMarketUpdated()
	return &MarketUpdated{
		Base: newBaseFromBusEvent(ctx, MarketUpdatedEvent, be),
		pm:   *m,
	}
}
