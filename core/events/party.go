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

type Party struct {
	*Base
	p types.Party
}

func NewPartyEvent(ctx context.Context, party types.Party) *Party {
	return &Party{
		Base: newBase(ctx, PartyEvent),
		p:    party,
	}
}

func (p Party) IsParty(id string) bool {
	return p.p.Id == id
}

func (p *Party) Party() types.Party {
	return p.p
}

func (p Party) Proto() *proto.Party {
	return p.p.IntoProto()
}

func (p Party) StreamMessage() *eventspb.BusEvent {
	busEvent := newBusEventFromBase(p.Base)
	busEvent.Event = &eventspb.BusEvent_Party{
		Party: p.Proto(),
	}

	return busEvent
}

func PartyEventFromStream(ctx context.Context, be *eventspb.BusEvent) *Party {
	return &Party{
		Base: newBaseFromBusEvent(ctx, PartyEvent, be),
		p:    types.Party{Id: be.GetParty().Id},
	}
}
