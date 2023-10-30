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
	"time"

	"code.vegaprotocol.io/vega/core/types"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

type ReferralProgramStarted struct {
	*Base
	e eventspb.ReferralProgramStarted
}

func (t ReferralProgramStarted) StreamMessage() *eventspb.BusEvent {
	busEvent := newBusEventFromBase(t.Base)
	busEvent.Event = &eventspb.BusEvent_ReferralProgramStarted{
		ReferralProgramStarted: &t.e,
	}

	return busEvent
}

func NewReferralProgramStartedEvent(ctx context.Context, p *types.ReferralProgram, epochTime time.Time, epoch uint64) *ReferralProgramStarted {
	return &ReferralProgramStarted{
		Base: newBase(ctx, ReferralProgramStartedEvent),
		e: eventspb.ReferralProgramStarted{
			Program:   p.IntoProto(),
			StartedAt: epochTime.UnixNano(),
			AtEpoch:   epoch,
		},
	}
}

func (r ReferralProgramStarted) GetReferralProgramStarted() *eventspb.ReferralProgramStarted {
	return &r.e
}

func ReferralProgramStartedEventFromStream(ctx context.Context, be *eventspb.BusEvent) *ReferralProgramStarted {
	return &ReferralProgramStarted{
		Base: newBaseFromBusEvent(ctx, ReferralProgramStartedEvent, be),
		e:    *be.GetReferralProgramStarted(),
	}
}

type ReferralProgramUpdated struct {
	*Base
	e eventspb.ReferralProgramUpdated
}

func (t ReferralProgramUpdated) StreamMessage() *eventspb.BusEvent {
	busEvent := newBusEventFromBase(t.Base)
	busEvent.Event = &eventspb.BusEvent_ReferralProgramUpdated{
		ReferralProgramUpdated: &t.e,
	}

	return busEvent
}

func (r ReferralProgramUpdated) GetReferralProgramUpdated() *eventspb.ReferralProgramUpdated {
	return &r.e
}

func NewReferralProgramUpdatedEvent(ctx context.Context, p *types.ReferralProgram, epochTime time.Time, epoch uint64) *ReferralProgramUpdated {
	return &ReferralProgramUpdated{
		Base: newBase(ctx, ReferralProgramUpdatedEvent),
		e: eventspb.ReferralProgramUpdated{
			Program:   p.IntoProto(),
			UpdatedAt: epochTime.UnixNano(),
			AtEpoch:   epoch,
		},
	}
}

func ReferralProgramUpdatedEventFromStream(ctx context.Context, be *eventspb.BusEvent) *ReferralProgramUpdated {
	return &ReferralProgramUpdated{
		Base: newBaseFromBusEvent(ctx, ReferralProgramUpdatedEvent, be),
		e:    *be.GetReferralProgramUpdated(),
	}
}

type ReferralProgramEnded struct {
	*Base
	e eventspb.ReferralProgramEnded
}

func (t ReferralProgramEnded) StreamMessage() *eventspb.BusEvent {
	busEvent := newBusEventFromBase(t.Base)
	busEvent.Event = &eventspb.BusEvent_ReferralProgramEnded{
		ReferralProgramEnded: &t.e,
	}

	return busEvent
}

func (t ReferralProgramEnded) GetReferralProgramEnded() *eventspb.ReferralProgramEnded {
	return &t.e
}

func NewReferralProgramEndedEvent(ctx context.Context, version uint64, id string, epochTime time.Time, epoch uint64) *ReferralProgramEnded {
	return &ReferralProgramEnded{
		Base: newBase(ctx, ReferralProgramEndedEvent),
		e: eventspb.ReferralProgramEnded{
			Version: version,
			Id:      id,
			EndedAt: epochTime.UnixNano(),
			AtEpoch: epoch,
		},
	}
}

func ReferralProgramEndedEventFromStream(ctx context.Context, be *eventspb.BusEvent) *ReferralProgramEnded {
	return &ReferralProgramEnded{
		Base: newBaseFromBusEvent(ctx, ReferralProgramEndedEvent, be),
		e:    *be.GetReferralProgramEnded(),
	}
}
