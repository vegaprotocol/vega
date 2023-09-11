package events

import (
	"context"

	"code.vegaprotocol.io/vega/core/types"
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

func NewVolumeDiscountProgramStartedEvent(ctx context.Context, p *types.VolumeDiscountProgram) *VolumeDiscountProgramStarted {
	return &VolumeDiscountProgramStarted{
		Base: newBase(ctx, VolumeDiscountProgramStartedEvent),
		e: &eventspb.VolumeDiscountProgramStarted{
			Program: p.IntoProto(),
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

func NewVolumeDiscountProgramUpdatedEvent(ctx context.Context, p *types.VolumeDiscountProgram) *VolumeDiscountProgramUpdated {
	return &VolumeDiscountProgramUpdated{
		Base: newBase(ctx, VolumeDiscountProgramUpdatedEvent),
		e: &eventspb.VolumeDiscountProgramUpdated{
			Program: p.IntoProto(),
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

func NewVolumeDiscountProgramEndedEvent(ctx context.Context, version uint64, id string) *VolumeDiscountProgramEnded {
	return &VolumeDiscountProgramEnded{
		Base: newBase(ctx, VolumeDiscountProgramEndedEvent),
		e: &eventspb.VolumeDiscountProgramEnded{
			Version: version,
			Id:      id,
		},
	}
}

func VolumeDiscountProgramEndedEventFromStream(ctx context.Context, be *eventspb.BusEvent) *VolumeDiscountProgramEnded {
	return &VolumeDiscountProgramEnded{
		Base: newBaseFromBusEvent(ctx, VolumeDiscountProgramEndedEvent, be),
		e:    be.GetVolumeDiscountProgramEnded(),
	}
}
