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

type EndBlock struct {
	*Base
	eb eventspb.EndBlock
}

// NewTime returns a new time Update event.
func NewEndBlock(ctx context.Context, bb eventspb.EndBlock) *EndBlock {
	return &EndBlock{
		Base: newBase(ctx, EndBlockEvent),
		eb:   bb,
	}
}

// Time returns the new blocktime.
func (e EndBlock) EndBlock() eventspb.EndBlock {
	return e.eb
}

func (e EndBlock) Proto() eventspb.EndBlock {
	return e.eb
}

func (e EndBlock) StreamMessage() *eventspb.BusEvent {
	p := e.Proto()
	busEvent := newBusEventFromBase(e.Base)
	busEvent.Event = &eventspb.BusEvent_EndBlock{
		EndBlock: &p,
	}

	return busEvent
}

func EndBlockEventFromStream(ctx context.Context, be *eventspb.BusEvent) *EndBlock {
	return &EndBlock{
		Base: newBaseFromBusEvent(ctx, EndBlockEvent, be),
		eb:   *be.GetEndBlock(),
	}
}
