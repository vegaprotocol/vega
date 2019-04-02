package vegatime

import (
	"time"

	types "code.vegaprotocol.io/vega/proto"
)

// UnixNano equivalent to time.Unix(sec, nsec) but to be used with the
// result of time.Time{}.UnixNano()
func UnixNano(nsec int64) time.Time {
	return time.Unix(nsec/int64(time.Second), nsec%int64(time.Second))
}

func RoundToNearest(t time.Time, interval types.Interval) time.Time {
	switch interval {
	case types.Interval_I1M:
		return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), 0, 0, t.Location())
	case types.Interval_I5M:
		return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), (t.Minute()/5)*5, 0, 0, t.Location())
	case types.Interval_I15M:
		return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), (t.Minute()/15)*15, 0, 0, t.Location())
	case types.Interval_I1H:
		return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), 0, 0, 0, t.Location())
	case types.Interval_I6H:
		return time.Date(t.Year(), t.Month(), t.Day(), (t.Hour()/6)*6, 0, 0, 0, t.Location())
	case types.Interval_I1D:
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	default:
		return t
	}
}
