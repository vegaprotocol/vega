package services

import "time"

func unixTimestamp(datetime time.Time) uint64 {
	return uint64(datetime.UnixNano() / (int64(time.Millisecond)/int64(time.Nanosecond)))
}
