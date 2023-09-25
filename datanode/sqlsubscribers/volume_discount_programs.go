package sqlsubscribers

import (
	"context"
	"time"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/datanode/entities"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

type (
	VolumeDiscountProgramStartedEvent interface {
		events.Event
		GetVolumeDiscountProgramStarted() *eventspb.VolumeDiscountProgramStarted
	}

	VolumeDiscountProgramUpdatedEvent interface {
		events.Event
		GetVolumeDiscountProgramUpdated() *eventspb.VolumeDiscountProgramUpdated
	}

	VolumeDiscountProgramEndedEvent interface {
		events.Event
		GetVolumeDiscountProgramEnded() *eventspb.VolumeDiscountProgramEnded
	}

	VolumeDiscountStore interface {
		AddVolumeDiscountProgram(ctx context.Context, referral *entities.VolumeDiscountProgram) error
		UpdateVolumeDiscountProgram(ctx context.Context, referral *entities.VolumeDiscountProgram) error
		EndVolumeDiscountProgram(ctx context.Context, version uint64, vegaTime time.Time, seqNum uint64) error
	}

	VolumeDiscountProgram struct {
		subscriber
		store VolumeDiscountStore
	}
)

func NewVolumeDiscountProgram(store VolumeDiscountStore) *VolumeDiscountProgram {
	return &VolumeDiscountProgram{
		store: store,
	}
}

func (rp *VolumeDiscountProgram) Types() []events.Type {
	return []events.Type{
		events.VolumeDiscountProgramStartedEvent,
		events.VolumeDiscountProgramUpdatedEvent,
		events.VolumeDiscountProgramEndedEvent,
	}
}

func (rp *VolumeDiscountProgram) Push(ctx context.Context, evt events.Event) error {
	switch e := evt.(type) {
	case VolumeDiscountProgramStartedEvent:
		return rp.consumeVolumeDiscountProgramStartedEvent(ctx, e)
	case VolumeDiscountProgramUpdatedEvent:
		return rp.consumeVolumeDiscountProgramUpdatedEvent(ctx, e)
	case VolumeDiscountProgramEndedEvent:
		return rp.consumeVolumeDiscountProgramEndedEvent(ctx, e)
	default:
		return nil
	}
}

func (rp *VolumeDiscountProgram) consumeVolumeDiscountProgramStartedEvent(ctx context.Context, e VolumeDiscountProgramStartedEvent) error {
	program := entities.VolumeDiscountProgramFromProto(e.GetVolumeDiscountProgramStarted().GetProgram(), rp.vegaTime, e.Sequence())
	return rp.store.AddVolumeDiscountProgram(ctx, program)
}

func (rp *VolumeDiscountProgram) consumeVolumeDiscountProgramUpdatedEvent(ctx context.Context, e VolumeDiscountProgramUpdatedEvent) error {
	program := entities.VolumeDiscountProgramFromProto(e.GetVolumeDiscountProgramUpdated().GetProgram(), rp.vegaTime, e.Sequence())
	return rp.store.UpdateVolumeDiscountProgram(ctx, program)
}

func (rp *VolumeDiscountProgram) consumeVolumeDiscountProgramEndedEvent(ctx context.Context, e VolumeDiscountProgramEndedEvent) error {
	return rp.store.EndVolumeDiscountProgram(ctx, e.GetVolumeDiscountProgramEnded().GetVersion(), rp.vegaTime, e.Sequence())
}
