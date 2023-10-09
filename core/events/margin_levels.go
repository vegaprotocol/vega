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

// MarginLevels - the margin levels event.
type MarginLevels struct {
	*Base
	l proto.MarginLevels
}

func NewMarginLevelsEvent(ctx context.Context, l types.MarginLevels) *MarginLevels {
	return &MarginLevels{
		Base: newBase(ctx, MarginLevelsEvent),
		l:    *l.IntoProto(),
	}
}

func (m MarginLevels) MarginLevels() proto.MarginLevels {
	return m.l
}

func (m MarginLevels) IsParty(id string) bool {
	return m.l.PartyId == id
}

func (m MarginLevels) PartyID() string {
	return m.l.PartyId
}

func (m MarginLevels) MarketID() string {
	return m.l.MarketId
}

func (m MarginLevels) Asset() string {
	return m.l.Asset
}

func (m MarginLevels) Proto() proto.MarginLevels {
	return m.l
}

func (m MarginLevels) StreamMessage() *eventspb.BusEvent {
	busEvent := newBusEventFromBase(m.Base)
	busEvent.Event = &eventspb.BusEvent_MarginLevels{
		MarginLevels: &m.l,
	}

	return busEvent
}

func MarginLevelsEventFromStream(ctx context.Context, be *eventspb.BusEvent) *MarginLevels {
	return &MarginLevels{
		Base: newBaseFromBusEvent(ctx, MarginLevelsEvent, be),
		l:    *be.GetMarginLevels(),
	}
}
