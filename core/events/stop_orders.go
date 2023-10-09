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

type StopOrder struct {
	*Base
	so *eventspb.StopOrderEvent
}

func NewStopOrderEvent(ctx context.Context, so *types.StopOrder) *StopOrder {
	stop := &StopOrder{
		Base: newBase(ctx, StopOrderEvent),
		so:   so.ToProtoEvent(),
	}

	return stop
}

func (o StopOrder) StopOrder() *eventspb.StopOrderEvent {
	return o.so
}

func (o StopOrder) StreamMessage() *eventspb.BusEvent {
	busEvent := newBusEventFromBase(o.Base)
	busEvent.Event = &eventspb.BusEvent_StopOrder{
		StopOrder: o.so,
	}
	return busEvent
}

func StopOrderEventFromStream(ctx context.Context, be *eventspb.BusEvent) *StopOrder {
	stop := &StopOrder{
		Base: newBaseFromBusEvent(ctx, StopOrderEvent, be),
		so:   be.GetStopOrder(),
	}
	return stop
}
