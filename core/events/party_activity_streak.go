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

	"code.vegaprotocol.io/vega/libs/ptr"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

type PartyActivityStreak struct {
	*Base
	pas eventspb.PartyActivityStreak
}

func NewPartyActivityStreakEvent(ctx context.Context, pas *eventspb.PartyActivityStreak) *PartyActivityStreak {
	order := &PartyActivityStreak{
		Base: newBase(ctx, PartyActivityStreakEvent),
		pas:  *pas,
	}
	return order
}

func (p *PartyActivityStreak) PartyActivityStreak() *eventspb.PartyActivityStreak {
	return ptr.From(p.pas)
}

func (p PartyActivityStreak) Proto() eventspb.PartyActivityStreak {
	return p.pas
}

func (p PartyActivityStreak) StreamMessage() *eventspb.BusEvent {
	busEvent := newBusEventFromBase(p.Base)
	busEvent.Event = &eventspb.BusEvent_PartyActivityStreak{
		PartyActivityStreak: ptr.From(p.pas),
	}

	return busEvent
}

func PartyActivityStreakEventFromStream(ctx context.Context, be *eventspb.BusEvent) *PartyActivityStreak {
	order := &PartyActivityStreak{
		Base: newBaseFromBusEvent(ctx, PartyActivityStreakEvent, be),
		pas:  ptr.UnBox(be.GetPartyActivityStreak()),
	}
	return order
}
