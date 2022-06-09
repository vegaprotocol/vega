package entities

import (
	"time"

	"code.vegaprotocol.io/protos/vega"
	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
)

type Epoch struct {
	ID         int64
	StartTime  time.Time
	ExpireTime time.Time
	EndTime    *time.Time
	VegaTime   time.Time
}

func (e *Epoch) ToProto() *vega.Epoch {
	protoEpoch := vega.Epoch{
		Seq: uint64(e.ID),
		Timestamps: &vega.EpochTimestamps{
			StartTime:  e.StartTime.UnixNano(),
			ExpiryTime: e.ExpireTime.UnixNano(),
		},
	}
	if e.EndTime != nil {
		protoEpoch.Timestamps.EndTime = e.EndTime.UnixNano()
	}
	return &protoEpoch
}

func EpochFromProto(ee eventspb.EpochEvent) Epoch {
	var endTime *time.Time
	if ee.Action == vega.EpochAction_EPOCH_ACTION_END {
		t := NanosToPostgresTimestamp(ee.EndTime)
		endTime = &t
	}
	epoch := Epoch{
		ID:         int64(ee.Seq),
		StartTime:  NanosToPostgresTimestamp(ee.StartTime),
		ExpireTime: NanosToPostgresTimestamp(ee.ExpireTime),
		EndTime:    endTime,
	}
	return epoch
}
