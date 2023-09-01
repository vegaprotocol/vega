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
