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

type EpochEvent struct {
	*Base
	e *eventspb.EpochEvent
}

func NewEpochEvent(ctx context.Context, e *types.Epoch) *EpochEvent {
	epoch := &EpochEvent{
		Base: newBase(ctx, EpochUpdate),
		e:    e.IntoProto(),
	}
	return epoch
}

func (e *EpochEvent) Epoch() *eventspb.EpochEvent {
	return e.e
}

func (e EpochEvent) Proto() eventspb.EpochEvent {
	return *e.e
}

func (e EpochEvent) StreamMessage() *eventspb.BusEvent {
	busEvent := newBusEventFromBase(e.Base)
	busEvent.Event = &eventspb.BusEvent_EpochEvent{
		EpochEvent: e.e,
	}
	return busEvent
}

func EpochEventFromStream(ctx context.Context, be *eventspb.BusEvent) *EpochEvent {
	return &EpochEvent{
		Base: newBaseFromBusEvent(ctx, EpochUpdate, be),
		e:    be.GetEpochEvent(),
	}
}
