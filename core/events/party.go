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
	"strings"

	"code.vegaprotocol.io/vega/core/types"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"

	"golang.org/x/exp/slices"
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

func (p Party) Proto() *vegapb.Party {
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

type PartyProfileUpdated struct {
	*Base
	e eventspb.PartyProfileUpdated
}

func (t PartyProfileUpdated) StreamMessage() *eventspb.BusEvent {
	busEvent := newBusEventFromBase(t.Base)
	busEvent.Event = &eventspb.BusEvent_PartyProfileUpdated{
		PartyProfileUpdated: &t.e,
	}

	return busEvent
}

func (t PartyProfileUpdated) PartyProfileUpdated() *eventspb.PartyProfileUpdated {
	return &t.e
}

func NewPartyProfileUpdatedEvent(ctx context.Context, p *types.PartyProfile) *PartyProfileUpdated {
	metadata := make([]*vegapb.Metadata, 0, len(p.Metadata))
	for k, v := range p.Metadata {
		metadata = append(metadata, &vegapb.Metadata{
			Key:   k,
			Value: v,
		})
	}

	// Ensure deterministic order in event.
	slices.SortStableFunc(metadata, func(a, b *vegapb.Metadata) int {
		return strings.Compare(a.Key, b.Key)
	})

	return &PartyProfileUpdated{
		Base: newBase(ctx, PartyProfileUpdatedEvent),
		e: eventspb.PartyProfileUpdated{
			UpdatedProfile: &vegapb.PartyProfile{
				PartyId:  p.PartyID.String(),
				Alias:    p.Alias,
				Metadata: metadata,
			},
		},
	}
}

func PartyProfileUpdatedEventFromStream(ctx context.Context, be *eventspb.BusEvent) *PartyProfileUpdated {
	return &PartyProfileUpdated{
		Base: newBaseFromBusEvent(ctx, PartyProfileUpdatedEvent, be),
		e:    *be.GetPartyProfileUpdated(),
	}
}
