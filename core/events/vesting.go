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

type VestingStatsUpdated struct {
	*Base
	vsu eventspb.VestingStatsUpdated
}

func NewVestingStatsUpdatedEvent(ctx context.Context, vsu *eventspb.VestingStatsUpdated) *VestingStatsUpdated {
	order := &VestingStatsUpdated{
		Base: newBase(ctx, VestingStatsUpdatedEvent),
		vsu:  *vsu,
	}
	return order
}

func (p *VestingStatsUpdated) VestingStatsUpdated() *eventspb.VestingStatsUpdated {
	return ptr.From(p.vsu)
}

func (p VestingStatsUpdated) Proto() eventspb.VestingStatsUpdated {
	return p.vsu
}

func (p VestingStatsUpdated) StreamMessage() *eventspb.BusEvent {
	busEvent := newBusEventFromBase(p.Base)
	busEvent.Event = &eventspb.BusEvent_VestingStatsUpdated{
		VestingStatsUpdated: ptr.From(p.vsu),
	}

	return busEvent
}

func VestingStatsUpdatedEventFromStream(ctx context.Context, be *eventspb.BusEvent) *VestingStatsUpdated {
	order := &VestingStatsUpdated{
		Base: newBaseFromBusEvent(ctx, VestingStatsUpdatedEvent, be),
		vsu:  ptr.UnBox(be.GetVestingStatsUpdated()),
	}
	return order
}
