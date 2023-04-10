package entities

import (
	"time"

	"code.vegaprotocol.io/vega/libs/ptr"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
)

type DateRange struct {
	Start *time.Time
	End   *time.Time
}

func DateRangeFromProto(dateRangeInput *v2.DateRange) (dateRange DateRange) {
	if dateRangeInput == nil {
		return
	}

	if dateRangeInput.StartTimestamp != nil {
		dateRange.Start = ptr.From(time.Unix(0, *dateRangeInput.StartTimestamp))
	}

	if dateRangeInput.EndTimestamp != nil {
		dateRange.End = ptr.From(time.Unix(0, *dateRangeInput.EndTimestamp))
	}

	return
}
