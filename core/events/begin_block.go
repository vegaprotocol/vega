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

type BeginBlock struct {
	*Base
	bb eventspb.BeginBlock
}

// NewTime returns a new time Update event.
func NewBeginBlock(ctx context.Context, bb eventspb.BeginBlock) *BeginBlock {
	return &BeginBlock{
		Base: newBase(ctx, BeginBlockEvent),
		bb:   bb,
	}
}

// Time returns the new blocktime.
func (b BeginBlock) BeginBlock() eventspb.BeginBlock {
	return b.bb
}

func (b BeginBlock) Proto() eventspb.BeginBlock {
	return b.bb
}

func (b BeginBlock) StreamMessage() *eventspb.BusEvent {
	p := b.Proto()
	busEvent := newBusEventFromBase(b.Base)
	busEvent.Event = &eventspb.BusEvent_BeginBlock{
		BeginBlock: &p,
	}

	return busEvent
}

func BeginBlockEventFromStream(ctx context.Context, be *eventspb.BusEvent) *BeginBlock {
	return &BeginBlock{
		Base: newBaseFromBusEvent(ctx, BeginBlockEvent, be),
		bb:   *be.GetBeginBlock(),
	}
}
