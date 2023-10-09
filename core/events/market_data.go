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

	"code.vegaprotocol.io/vega/core/types"
	proto "code.vegaprotocol.io/vega/protos/vega"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

type MarketData struct {
	*Base
	md proto.MarketData
}

func NewMarketDataEvent(ctx context.Context, md types.MarketData) *MarketData {
	return &MarketData{
		Base: newBase(ctx, MarketDataEvent),
		md:   *md.IntoProto(),
	}
}

func (m MarketData) MarketID() string {
	return m.md.Market
}

func (m MarketData) MarketData() proto.MarketData {
	return m.md
}

func (m MarketData) Proto() proto.MarketData {
	return m.md
}

func (m MarketData) StreamMessage() *eventspb.BusEvent {
	busEvent := newBusEventFromBase(m.Base)
	busEvent.Event = &eventspb.BusEvent_MarketData{
		MarketData: &m.md,
	}

	return busEvent
}

func MarketDataEventFromStream(ctx context.Context, be *eventspb.BusEvent) *MarketData {
	return &MarketData{
		Base: newBaseFromBusEvent(ctx, MarketDataEvent, be),
		md:   *be.GetMarketData(),
	}
}
