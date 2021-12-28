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
	return &eventspb.BusEvent{
		Version: eventspb.Version,
		Id:      e.eventID(),
		Block:   e.TraceID(),
		ChainId: e.ChainID(),
		Type:    e.et.ToProto(),
		Event: &eventspb.BusEvent_EpochEvent{
			EpochEvent: e.e,
		},
	}
}

func EpochEventFromStream(ctx context.Context, be *eventspb.BusEvent) *EpochEvent {
	return &EpochEvent{
		Base: newBaseFromStream(ctx, EpochUpdate, be),
		e:    be.GetEpochEvent(),
	}
}
