package initialise

import (
	"time"

	"code.vegaprotocol.io/vega/datanode/config/encoding"
)

type Config struct {
	MinimumBlockCount int64             `long:"block-count" description:"the minimum number of blocks to fetch"`
	TimeOut           encoding.Duration `long:"timeout" description:"maximum time allowed to auto-initialise the node"`
}

func NewDefaultConfig() Config {
	return Config{
		MinimumBlockCount: 1,
		TimeOut:           encoding.Duration{Duration: 1 * time.Minute},
	}
}
