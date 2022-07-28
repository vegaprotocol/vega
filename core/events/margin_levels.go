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

	proto "code.vegaprotocol.io/protos/vega"
	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	"code.vegaprotocol.io/vega/core/types"
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
