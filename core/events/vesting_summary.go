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

	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

type VestingBalancesSummary struct {
	*Base
	vbs *eventspb.VestingBalancesSummary
}

func NewVestingBalancesSummaryEvent(ctx context.Context, vbs *eventspb.VestingBalancesSummary) *VestingBalancesSummary {
	order := &VestingBalancesSummary{
		Base: newBase(ctx, VestingBalancesSummaryEvent),
		vbs:  vbs,
	}
	return order
}

func (v *VestingBalancesSummary) VestingBalancesSummary() *eventspb.VestingBalancesSummary {
	return v.vbs
}

func (v VestingBalancesSummary) Proto() eventspb.VestingBalancesSummary {
	return *v.vbs
}

func (v VestingBalancesSummary) StreamMessage() *eventspb.BusEvent {
	busEvent := newBusEventFromBase(v.Base)
	busEvent.Event = &eventspb.BusEvent_VestingBalancesSummary{
		VestingBalancesSummary: v.vbs,
	}

	return busEvent
}

func VestingBalancesSummaryEventFromStream(ctx context.Context, be *eventspb.BusEvent) *VestingBalancesSummary {
	order := &VestingBalancesSummary{
		Base: newBaseFromBusEvent(ctx, VestingBalancesSummaryEvent, be),
		vbs:  be.GetVestingBalancesSummary(),
	}
	return order
}
