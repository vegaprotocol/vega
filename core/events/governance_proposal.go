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

type Proposal struct {
	*Base
	p proto.Proposal
}

func NewProposalEvent(ctx context.Context, p types.Proposal) *Proposal {
	ip := p.IntoProto()
	return &Proposal{
		Base: newBase(ctx, ProposalEvent),
		p:    *ip,
	}
}

func (p *Proposal) Proposal() proto.Proposal {
	return p.p
}

// ProposalID - for combined subscriber, communal interface.
func (p *Proposal) ProposalID() string {
	return p.p.Id
}

func (p Proposal) IsParty(id string) bool {
	return p.p.PartyId == id
}

// PartyID - for combined subscriber, communal interface.
func (p *Proposal) PartyID() string {
	return p.p.PartyId
}

func (p Proposal) Proto() proto.Proposal {
	return p.p
}

func (p Proposal) StreamMessage() *eventspb.BusEvent {
	busEvent := newBusEventFromBase(p.Base)
	busEvent.Event = &eventspb.BusEvent_Proposal{
		Proposal: &p.p,
	}
	return busEvent
}

func ProposalEventFromStream(ctx context.Context, be *eventspb.BusEvent) *Proposal {
	return &Proposal{
		Base: newBaseFromBusEvent(ctx, ProposalEvent, be),
		p:    *be.GetProposal(),
	}
}
