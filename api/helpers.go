package api

import (
	"time"
	"github.com/satori/go.uuid"
)

func unixTimestamp(datetime time.Time) uint64 {
	return uint64(datetime.UnixNano() / (int64(time.Millisecond)/int64(time.Nanosecond)))
}

func newGuid() string {
	return uuid.NewV4().String()
}

func bytesWithPipedGuid(input []byte) ([]byte, error) {
	prefix := newGuid() + "|"
	prefixBytes := []byte(prefix)
	return append(prefixBytes, input...), nil
}