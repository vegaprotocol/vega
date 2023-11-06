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
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

type StakeLinking struct {
	*Base
	evt eventspb.StakeLinking
}

func NewStakeLinking(ctx context.Context, evt types.StakeLinking) *StakeLinking {
	return &StakeLinking{
		Base: newBase(ctx, StakeLinkingEvent),
		evt:  *(evt.IntoProto()),
	}
}

func (s StakeLinking) StakeLinking() eventspb.StakeLinking {
	return s.evt
}

func (s StakeLinking) Proto() eventspb.StakeLinking {
	return s.evt
}

func (s StakeLinking) StreamMessage() *eventspb.BusEvent {
	busEvent := newBusEventFromBase(s.Base)
	busEvent.Event = &eventspb.BusEvent_StakeLinking{
		StakeLinking: &s.evt,
	}

	return busEvent
}

func StakeLinkingFromStream(ctx context.Context, be *eventspb.BusEvent) *StakeLinking {
	return &StakeLinking{
		Base: newBaseFromBusEvent(ctx, StakeLinkingEvent, be),
		evt:  *be.GetStakeLinking(),
	}
}
