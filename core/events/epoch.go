// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
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

	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	"code.vegaprotocol.io/vega/types"
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
