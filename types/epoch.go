//lint:file-ignore ST1003 Ignore underscores in names, this is straigh copied from the proto package to ease introducing the domain types

package types

import (
	"fmt"
	"time"

	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
)

type Epoch struct {
	// Unique identifier that increases by one each epoch
	Seq uint64
	// What time did this epoch start
	StartTime time.Time
	// What time should this epoch end
	ExpireTime time.Time
	// What time did it actually end
	EndTime time.Time
}

func (e *Epoch) String() string {
	return fmt.Sprintln("Seq", e.Seq, "StartTime", e.StartTime,
		"ExpireTime", e.ExpireTime, "EndTime", e.EndTime)
}

func NewEpochFromProto(p *eventspb.EpochEvent) *Epoch {
	e := &Epoch{
		Seq:        p.Seq,
		StartTime:  time.Unix(0, p.StartTime),
		ExpireTime: time.Unix(0, p.ExpireTime),
		EndTime:    time.Unix(0, p.EndTime),
	}
	return e
}

func (e Epoch) IntoProto() *eventspb.EpochEvent {
	return &eventspb.EpochEvent{
		Seq:        e.Seq,
		StartTime:  e.StartTime.UnixNano(),
		ExpireTime: e.ExpireTime.UnixNano(),
		EndTime:    e.EndTime.UnixNano(),
	}
}
