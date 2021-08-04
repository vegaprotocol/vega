package events

import (
	"context"

	eventspb "code.vegaprotocol.io/protos/vega/events/v1"

	"code.vegaprotocol.io/vega/types"
)

type StakingEvt struct {
	*Base
	evt eventspb.StakingEvent
}

func NewStakingEvent(ctx context.Context, evt types.StakingEvent) *StakingEvt {
	return &StakingEvt{
		Base: newBase(ctx, StakingEvent),
		evt:  *(evt.IntoProto()),
	}
}

func (s *StakingEvt) StakingEvtount() eventspb.StakingEvent {
	return s.evt
}

func (s StakingEvt) Proto() eventspb.StakingEvent {
	return s.evt
}

func (s StakingEvt) StreamMessage() *eventspb.BusEvent {
	return &eventspb.BusEvent{
		Id:    s.eventID(),
		Block: s.TraceID(),
		Type:  s.et.ToProto(),
		Event: &eventspb.BusEvent_StakingEvent{
			StakingEvent: &s.evt,
		},
	}
}
