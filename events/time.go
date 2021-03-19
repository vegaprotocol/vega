package events

import (
	"context"
	"time"

	eventspb "code.vegaprotocol.io/vega/proto/events/v1"
)

// Time event indicating a change in block time (ie time update)
type Time struct {
	*Base
	blockTime time.Time
}

// NewTime returns a new time Update event
func NewTime(ctx context.Context, t time.Time) *Time {
	return &Time{
		Base:      newBase(ctx, TimeUpdate),
		blockTime: t,
	}
}

// Time returns the new blocktime
func (t Time) Time() time.Time {
	return t.blockTime
}

func (t Time) Proto() eventspb.TimeUpdate {
	return eventspb.TimeUpdate{
		Timestamp: t.blockTime.UTC().Unix(),
	}
}

func (t Time) StreamMessage() *eventspb.BusEvent {
	p := t.Proto()
	return &eventspb.BusEvent{
		Id:    t.eventID(),
		Block: t.TraceID(),
		Type:  t.et.ToProto(),
		Event: &eventspb.BusEvent_TimeUpdate{
			TimeUpdate: &p,
		},
	}
}
