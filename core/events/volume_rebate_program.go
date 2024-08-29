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

type VolumeRebateProgramStarted struct {
	*Base
	e *eventspb.VolumeRebateProgramStarted
}

func (v *VolumeRebateProgramStarted) GetVolumeRebateProgramStarted() *eventspb.VolumeRebateProgramStarted {
	return v.e
}

func (t *VolumeRebateProgramStarted) StreamMessage() *eventspb.BusEvent {
	busEvent := newBusEventFromBase(t.Base)
	busEvent.Event = &eventspb.BusEvent_VolumeRebateProgramStarted{
		VolumeRebateProgramStarted: t.e,
	}

	return busEvent
}

func NewVolumeRebateProgramStartedEvent(ctx context.Context, p *types.VolumeRebateProgram, epochTime time.Time, epoch uint64) *VolumeRebateProgramStarted {
	return &VolumeRebateProgramStarted{
		Base: newBase(ctx, VolumeRebateProgramStartedEvent),
		e: &eventspb.VolumeRebateProgramStarted{
			Program:   p.IntoProto(),
			StartedAt: epochTime.UnixNano(),
			AtEpoch:   epoch,
		},
	}
}

func VolumeRebateProgramStartedEventFromStream(ctx context.Context, be *eventspb.BusEvent) *VolumeRebateProgramStarted {
	return &VolumeRebateProgramStarted{
		Base: newBaseFromBusEvent(ctx, VolumeRebateProgramStartedEvent, be),
		e:    be.GetVolumeRebateProgramStarted(),
	}
}

type VolumeRebateProgramUpdated struct {
	*Base
	e *eventspb.VolumeRebateProgramUpdated
}

func (v *VolumeRebateProgramUpdated) GetVolumeRebateProgramUpdated() *eventspb.VolumeRebateProgramUpdated {
	return v.e
}

func (t *VolumeRebateProgramUpdated) StreamMessage() *eventspb.BusEvent {
	busEvent := newBusEventFromBase(t.Base)
	busEvent.Event = &eventspb.BusEvent_VolumeRebateProgramUpdated{
		VolumeRebateProgramUpdated: t.e,
	}

	return busEvent
}

func NewVolumeRebateProgramUpdatedEvent(ctx context.Context, p *types.VolumeRebateProgram, epochTime time.Time, epoch uint64) *VolumeRebateProgramUpdated {
	return &VolumeRebateProgramUpdated{
		Base: newBase(ctx, VolumeRebateProgramUpdatedEvent),
		e: &eventspb.VolumeRebateProgramUpdated{
			Program:   p.IntoProto(),
			UpdatedAt: epochTime.UnixNano(),
			AtEpoch:   epoch,
		},
	}
}

func VolumeRebateProgramUpdatedEventFromStream(ctx context.Context, be *eventspb.BusEvent) *VolumeRebateProgramUpdated {
	return &VolumeRebateProgramUpdated{
		Base: newBaseFromBusEvent(ctx, VolumeRebateProgramUpdatedEvent, be),
		e:    be.GetVolumeRebateProgramUpdated(),
	}
}

type VolumeRebateProgramEnded struct {
	*Base
	e *eventspb.VolumeRebateProgramEnded
}

func (v *VolumeRebateProgramEnded) GetVolumeRebateProgramEnded() *eventspb.VolumeRebateProgramEnded {
	return v.e
}

func (t *VolumeRebateProgramEnded) StreamMessage() *eventspb.BusEvent {
	busEvent := newBusEventFromBase(t.Base)
	busEvent.Event = &eventspb.BusEvent_VolumeRebateProgramEnded{
		VolumeRebateProgramEnded: t.e,
	}

	return busEvent
}

func NewVolumeRebateProgramEndedEvent(ctx context.Context, version uint64, id string, epochTime time.Time, epoch uint64) *VolumeRebateProgramEnded {
	return &VolumeRebateProgramEnded{
		Base: newBase(ctx, VolumeRebateProgramEndedEvent),
		e: &eventspb.VolumeRebateProgramEnded{
			Version: version,
			Id:      id,
			EndedAt: epochTime.UnixNano(),
			AtEpoch: epoch,
		},
	}
}

func VolumeRebateProgramEndedEventFromStream(ctx context.Context, be *eventspb.BusEvent) *VolumeRebateProgramEnded {
	return &VolumeRebateProgramEnded{
		Base: newBaseFromBusEvent(ctx, VolumeRebateProgramEndedEvent, be),
		e:    be.GetVolumeRebateProgramEnded(),
	}
}

type VolumeRebateStatsUpdated struct {
	*Base
	vdsu eventspb.VolumeRebateStatsUpdated
}

func NewVolumeRebateStatsUpdatedEvent(ctx context.Context, vdsu *eventspb.VolumeRebateStatsUpdated) *VolumeRebateStatsUpdated {
	order := &VolumeRebateStatsUpdated{
		Base: newBase(ctx, VolumeRebateStatsUpdatedEvent),
		vdsu: *vdsu,
	}
	return order
}

func (p *VolumeRebateStatsUpdated) VolumeRebateStatsUpdated() *eventspb.VolumeRebateStatsUpdated {
	return ptr.From(p.vdsu)
}

func (p VolumeRebateStatsUpdated) Proto() eventspb.VolumeRebateStatsUpdated {
	return p.vdsu
}

func (p VolumeRebateStatsUpdated) StreamMessage() *eventspb.BusEvent {
	busEvent := newBusEventFromBase(p.Base)
	busEvent.Event = &eventspb.BusEvent_VolumeRebateStatsUpdated{
		VolumeRebateStatsUpdated: ptr.From(p.vdsu),
	}

	return busEvent
}

func VolumeRebateStatsUpdatedEventFromStream(ctx context.Context, be *eventspb.BusEvent) *VolumeRebateStatsUpdated {
	order := &VolumeRebateStatsUpdated{
		Base: newBaseFromBusEvent(ctx, VolumeRebateStatsUpdatedEvent, be),
		vdsu: ptr.UnBox(be.GetVolumeRebateStatsUpdated()),
	}
	return order
}
