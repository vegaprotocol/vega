package vegatime

import "time"

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


