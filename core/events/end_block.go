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
