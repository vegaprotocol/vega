package vegatime

import (
	"time"
	"vega/msg"
)

type Stamp uint64

func (s Stamp) Seconds() int64 {
	if s > 0 {
		secs := uint64(s) / uint64(1000 * time.Millisecond)
		return int64(secs)
	}
	return 0
}

func (s Stamp) NanoSeconds() int64 {
	secs := s.Seconds()
	if secs > 0 {
		nanoRemaining := uint64(s) % uint64(secs)
		return int64(nanoRemaining)
	}
	return 0
}

func (s Stamp) Rfc3339Nano() string {
	unixUtc := time.Unix(s.Seconds(), s.NanoSeconds())
	return unixUtc.Format(time.RFC3339Nano)
}

func (s Stamp) Rfc3339() string {
	unixUtc := time.Unix(s.Seconds(), s.NanoSeconds())
	return unixUtc.Format(time.RFC3339)
}

func (s Stamp) UnixNano() uint64 {
	return uint64(s)
}

func (s Stamp) Datetime() time.Time {
	return time.Unix(s.Seconds(), s.NanoSeconds())
}

func (s Stamp) RoundToNearest(interval msg.Interval) Stamp {
	t := s.Datetime()
	switch interval {
	case msg.Interval_I1M:
		return Stamp(uint64(time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), 0, 0, t.Location()).UnixNano()))
	case msg.Interval_I5M:
		return Stamp(uint64(time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), (t.Minute()/5)*5, 0, 0, t.Location()).UnixNano()))
	case msg.Interval_I15M:
		return Stamp(uint64(time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), (t.Minute()/15)*15, 0, 0, t.Location()).UnixNano()))
	case msg.Interval_I1H:
		return Stamp(uint64(time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), 0, 0, 0, t.Location()).UnixNano()))
	case msg.Interval_I6H:
		return Stamp(uint64(time.Date(t.Year(), t.Month(), t.Day(), (t.Hour()/6)*6, 0, 0, 0, t.Location()).UnixNano()))
	case msg.Interval_I1D:
		return Stamp(uint64(time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location()).UnixNano()))
	}

	return s
}