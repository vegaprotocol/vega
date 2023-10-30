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
