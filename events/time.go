package events

import (
	"context"
	"time"

	types "code.vegaprotocol.io/vega/proto"
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

func (t Time) Proto() types.TimeUpdate {
	return types.TimeUpdate{
		Timestamp: t.blockTime.UTC().Unix(),
	}
}

func (t Time) StreamMessage() *types.BusEvent {
	p := t.Proto()
	return &types.BusEvent{
		ID:   t.eventID(),
		Type: t.et.ToProto(),
		Event: &types.BusEvent_TimeUpdate{
			TimeUpdate: &p,
		},
	}
}
