package entities

import (
	"time"

	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
)

type DateRange struct {
	Start *time.Time
	End   *time.Time
}

func DateRangeFromProto(dateRange *v2.DateRange) DateRange {
	var startDate, endDate *time.Time

	if dateRange != nil && dateRange.StartTimestamp != nil {
		sd := time.Unix(0, *dateRange.StartTimestamp)
		startDate = &sd
	}

	if dateRange != nil && dateRange.EndTimestamp != nil {
		ed := time.Unix(0, *dateRange.EndTimestamp)
		endDate = &ed
	}

	return DateRange{
		Start: startDate,
		End:   endDate,
	}
}
