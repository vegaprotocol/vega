package events

import (
	"context"
	"time"

	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
)

// Time event indicating a change in block time (ie time update).
type Time struct {
	*Base
	blockTime time.Time
}

// NewTime returns a new time Update event.
func NewTime(ctx context.Context, t time.Time) *Time {
	return &Time{
		Base:      newBase(ctx, TimeUpdate),
		blockTime: t,
	}
}

// Time returns the new blocktime.
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
		Version: eventspb.Version,
		Id:      t.eventID(),
		Block:   t.TraceID(),
		Type:    t.et.ToProto(),
		Event: &eventspb.BusEvent_TimeUpdate{
			TimeUpdate: &p,
		},
	}
}

func TimeEventFromStream(ctx context.Context, be *eventspb.BusEvent) *Time {
	return &Time{
		Base:      newBaseFromStream(ctx, TimeUpdate, be),
		blockTime: time.Unix(be.GetTimeUpdate().Timestamp, 0).UTC(),
	}
}
