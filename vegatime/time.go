package vegatime

import (
	"time"

	types "code.vegaprotocol.io/vega/proto/gen/golang"
)

var now func() time.Time

func init() {
	now = time.Now
}

// SetNowFunc exists for testing purpose
// e.g: set the vegatime.Now function to return a specific time
//   vegatime.SetNowFunc(func() time.T { vegatime.Unix(123423, 0) })
// reset the vegatime.Now function
//   vegatime.SetNowFunc(nil)
// this will reset the vegatime.Now function to use time.Now() again
func SetNowFunc(f func() time.Time) {
	if f == nil {
		now = time.Now
	} else {
		now = f
	}
}

// Now return the current time in UTC timezone
func Now() time.Time {
	return now().UTC()
}

// Unix create a new time from sec and nsec in UTC timezone
func Unix(sec int64, nsec int64) time.Time {
	return time.Unix(sec, nsec).UTC()
}

// UnixNano equivalent to time.Unix(sec, nsec) but to be used with the
// result of time.Time{}.UnixNano() in UTC timezone
func UnixNano(nsec int64) time.Time {
	return time.Unix(nsec/int64(time.Second), nsec%int64(time.Second)).UTC()
}

// Parse parse a string expected to be a time in the time.RFC3339Nano format
func Parse(t string) (time.Time, error) {
	return time.ParseInLocation(time.RFC3339Nano, t, time.UTC)
}

// Format format the time using the time.RFC3339Nano formatting
func Format(t time.Time) string {
	return t.In(time.UTC).Format(time.RFC3339Nano)
}

// RoundToNearest round an actual time to the nearest interval
func RoundToNearest(t time.Time, interval types.Interval) time.Time {
	switch interval {
	case types.Interval_INTERVAL_I1M:
		return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), 0, 0, time.UTC)
	case types.Interval_INTERVAL_I5M:
		return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), (t.Minute()/5)*5, 0, 0, time.UTC)
	case types.Interval_INTERVAL_I15M:
		return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), (t.Minute()/15)*15, 0, 0, time.UTC)
	case types.Interval_INTERVAL_I1H:
		return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), 0, 0, 0, time.UTC)
	case types.Interval_INTERVAL_I6H:
		return time.Date(t.Year(), t.Month(), t.Day(), (t.Hour()/6)*6, 0, 0, 0, time.UTC)
	case types.Interval_INTERVAL_I1D:
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
	default:
		return t
	}
}
