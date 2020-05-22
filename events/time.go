package events

import (
	"context"
	"time"
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
