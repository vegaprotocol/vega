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

package sqlsubscribers

import (
	"context"
	"time"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/datanode/entities"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

type (
	VolumeRebateProgramStartedEvent interface {
		events.Event
		GetVolumeRebateProgramStarted() *eventspb.VolumeRebateProgramStarted
	}

	VolumeRebateProgramUpdatedEvent interface {
		events.Event
		GetVolumeRebateProgramUpdated() *eventspb.VolumeRebateProgramUpdated
	}

	VolumeRebateProgramEndedEvent interface {
		events.Event
		GetVolumeRebateProgramEnded() *eventspb.VolumeRebateProgramEnded
	}

	VolumeRebateStore interface {
		AddVolumeRebateProgram(ctx context.Context, referral *entities.VolumeRebateProgram) error
		UpdateVolumeRebateProgram(ctx context.Context, referral *entities.VolumeRebateProgram) error
		EndVolumeRebateProgram(ctx context.Context, version uint64, endedAt time.Time, vegaTime time.Time, seqNum uint64) error
	}

	VolumeRebateProgram struct {
		subscriber
		store VolumeRebateStore
	}
)

func NewVolumeRebateProgram(store VolumeRebateStore) *VolumeRebateProgram {
	return &VolumeRebateProgram{
		store: store,
	}
}

func (rp *VolumeRebateProgram) Types() []events.Type {
	return []events.Type{
		events.VolumeRebateProgramStartedEvent,
		events.VolumeRebateProgramUpdatedEvent,
		events.VolumeRebateProgramEndedEvent,
	}
}

func (rp *VolumeRebateProgram) Push(ctx context.Context, evt events.Event) error {
	switch e := evt.(type) {
	case VolumeRebateProgramStartedEvent:
		return rp.consumeVolumeRebateProgramStartedEvent(ctx, e)
	case VolumeRebateProgramUpdatedEvent:
		return rp.consumeVolumeRebateProgramUpdatedEvent(ctx, e)
	case VolumeRebateProgramEndedEvent:
		return rp.consumeVolumeRebateProgramEndedEvent(ctx, e)
	default:
		return nil
	}
}

func (rp *VolumeRebateProgram) consumeVolumeRebateProgramStartedEvent(ctx context.Context, e VolumeRebateProgramStartedEvent) error {
	program := entities.VolumeRebateProgramFromProto(e.GetVolumeRebateProgramStarted().GetProgram(), rp.vegaTime, e.Sequence())
	return rp.store.AddVolumeRebateProgram(ctx, program)
}

func (rp *VolumeRebateProgram) consumeVolumeRebateProgramUpdatedEvent(ctx context.Context, e VolumeRebateProgramUpdatedEvent) error {
	program := entities.VolumeRebateProgramFromProto(e.GetVolumeRebateProgramUpdated().GetProgram(), rp.vegaTime, e.Sequence())
	return rp.store.UpdateVolumeRebateProgram(ctx, program)
}

func (rp *VolumeRebateProgram) consumeVolumeRebateProgramEndedEvent(ctx context.Context, e VolumeRebateProgramEndedEvent) error {
	ev := e.GetVolumeRebateProgramEnded()
	return rp.store.EndVolumeRebateProgram(ctx, ev.GetVersion(), time.Unix(0, ev.EndedAt), rp.vegaTime, e.Sequence())
}

func (rp *VolumeRebateProgram) Name() string {
	return "VolumeRebateProgram"
}
