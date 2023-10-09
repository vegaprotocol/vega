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
	"code.vegaprotocol.io/vega/libs/ptr"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

type VolumeDiscountProgramStarted struct {
	*Base
	e *eventspb.VolumeDiscountProgramStarted
}

func (v *VolumeDiscountProgramStarted) GetVolumeDiscountProgramStarted() *eventspb.VolumeDiscountProgramStarted {
	return v.e
}

func (t *VolumeDiscountProgramStarted) StreamMessage() *eventspb.BusEvent {
	busEvent := newBusEventFromBase(t.Base)
	busEvent.Event = &eventspb.BusEvent_VolumeDiscountProgramStarted{
		VolumeDiscountProgramStarted: t.e,
	}

	return busEvent
}

func NewVolumeDiscountProgramStartedEvent(ctx context.Context, p *types.VolumeDiscountProgram, epochTime time.Time, epoch uint64) *VolumeDiscountProgramStarted {
	return &VolumeDiscountProgramStarted{
		Base: newBase(ctx, VolumeDiscountProgramStartedEvent),
		e: &eventspb.VolumeDiscountProgramStarted{
			Program:   p.IntoProto(),
			StartedAt: epochTime.UnixNano(),
			AtEpoch:   epoch,
		},
	}
}

func VolumeDiscountProgramStartedEventFromStream(ctx context.Context, be *eventspb.BusEvent) *VolumeDiscountProgramStarted {
	return &VolumeDiscountProgramStarted{
		Base: newBaseFromBusEvent(ctx, VolumeDiscountProgramStartedEvent, be),
		e:    be.GetVolumeDiscountProgramStarted(),
	}
}

type VolumeDiscountProgramUpdated struct {
	*Base
	e *eventspb.VolumeDiscountProgramUpdated
}

func (v *VolumeDiscountProgramUpdated) GetVolumeDiscountProgramUpdated() *eventspb.VolumeDiscountProgramUpdated {
	return v.e
}

func (t *VolumeDiscountProgramUpdated) StreamMessage() *eventspb.BusEvent {
	busEvent := newBusEventFromBase(t.Base)
	busEvent.Event = &eventspb.BusEvent_VolumeDiscountProgramUpdated{
		VolumeDiscountProgramUpdated: t.e,
	}

	return busEvent
}

func NewVolumeDiscountProgramUpdatedEvent(ctx context.Context, p *types.VolumeDiscountProgram, epochTime time.Time, epoch uint64) *VolumeDiscountProgramUpdated {
	return &VolumeDiscountProgramUpdated{
		Base: newBase(ctx, VolumeDiscountProgramUpdatedEvent),
		e: &eventspb.VolumeDiscountProgramUpdated{
			Program:   p.IntoProto(),
			UpdatedAt: epochTime.UnixNano(),
			AtEpoch:   epoch,
		},
	}
}

func VolumeDiscountProgramUpdatedEventFromStream(ctx context.Context, be *eventspb.BusEvent) *VolumeDiscountProgramUpdated {
	return &VolumeDiscountProgramUpdated{
		Base: newBaseFromBusEvent(ctx, VolumeDiscountProgramUpdatedEvent, be),
		e:    be.GetVolumeDiscountProgramUpdated(),
	}
}

type VolumeDiscountProgramEnded struct {
	*Base
	e *eventspb.VolumeDiscountProgramEnded
}

func (v *VolumeDiscountProgramEnded) GetVolumeDiscountProgramEnded() *eventspb.VolumeDiscountProgramEnded {
	return v.e
}

func (t *VolumeDiscountProgramEnded) StreamMessage() *eventspb.BusEvent {
	busEvent := newBusEventFromBase(t.Base)
	busEvent.Event = &eventspb.BusEvent_VolumeDiscountProgramEnded{
		VolumeDiscountProgramEnded: t.e,
	}

	return busEvent
}

func NewVolumeDiscountProgramEndedEvent(ctx context.Context, version uint64, id string, epochTime time.Time, epoch uint64) *VolumeDiscountProgramEnded {
	return &VolumeDiscountProgramEnded{
		Base: newBase(ctx, VolumeDiscountProgramEndedEvent),
		e: &eventspb.VolumeDiscountProgramEnded{
			Version: version,
			Id:      id,
			EndedAt: epochTime.UnixNano(),
			AtEpoch: epoch,
		},
	}
}

func VolumeDiscountProgramEndedEventFromStream(ctx context.Context, be *eventspb.BusEvent) *VolumeDiscountProgramEnded {
	return &VolumeDiscountProgramEnded{
		Base: newBaseFromBusEvent(ctx, VolumeDiscountProgramEndedEvent, be),
		e:    be.GetVolumeDiscountProgramEnded(),
	}
}

type VolumeDiscountStatsUpdated struct {
	*Base
	vdsu eventspb.VolumeDiscountStatsUpdated
}

func NewVolumeDiscountStatsUpdatedEvent(ctx context.Context, vdsu *eventspb.VolumeDiscountStatsUpdated) *VolumeDiscountStatsUpdated {
	order := &VolumeDiscountStatsUpdated{
		Base: newBase(ctx, VolumeDiscountStatsUpdatedEvent),
		vdsu: *vdsu,
	}
	return order
}

func (p *VolumeDiscountStatsUpdated) VolumeDiscountStatsUpdated() *eventspb.VolumeDiscountStatsUpdated {
	return ptr.From(p.vdsu)
}

func (p VolumeDiscountStatsUpdated) Proto() eventspb.VolumeDiscountStatsUpdated {
	return p.vdsu
}

func (p VolumeDiscountStatsUpdated) StreamMessage() *eventspb.BusEvent {
	busEvent := newBusEventFromBase(p.Base)
	busEvent.Event = &eventspb.BusEvent_VolumeDiscountStatsUpdated{
		VolumeDiscountStatsUpdated: ptr.From(p.vdsu),
	}

	return busEvent
}

func VolumeDiscountStatsUpdatedEventFromStream(ctx context.Context, be *eventspb.BusEvent) *VolumeDiscountStatsUpdated {
	order := &VolumeDiscountStatsUpdated{
		Base: newBaseFromBusEvent(ctx, VolumeDiscountStatsUpdatedEvent, be),
		vdsu: ptr.UnBox(be.GetVolumeDiscountStatsUpdated()),
	}
	return order
}
