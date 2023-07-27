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
