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
