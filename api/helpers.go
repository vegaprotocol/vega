package api

import (
	"time"
	"github.com/satori/go.uuid"
	"context"
)

func unixTimestamp(datetime time.Time) uint64 {
	return uint64(datetime.UnixNano() / (int64(time.Millisecond)/int64(time.Nanosecond)))
}

func newGuid() string {
	return uuid.NewV4().String()
}

func ipAddressFromContext(ctx context.Context) interface{} {
	return ctx.Value("remote-ip-addr")
}
