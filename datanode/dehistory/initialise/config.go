package initialise

import (
	"time"

	"code.vegaprotocol.io/vega/datanode/config/encoding"
)

type Config struct {
	MinimumBlockCount int64             `long:"block-count" description:"the minimum number of blocks to fetch"`
	TimeOut           encoding.Duration `long:"timeout" description:"maximum time allowed to auto-initialise the node"`
	GrpcAPIPorts      []int             `long:"grpc-api-ports" description:"list of additional ports to check to for api connection when getting latest segment"`
}

func NewDefaultConfig() Config {
	return Config{
		MinimumBlockCount: 1,
		TimeOut:           encoding.Duration{Duration: 1 * time.Minute},
		GrpcAPIPorts:      []int{},
	}
}
