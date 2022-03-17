package candlesv2

import (
	"time"

	"code.vegaprotocol.io/data-node/config/encoding"
	"code.vegaprotocol.io/data-node/logging"
)

// namedLogger is the identifier for package and should ideally match the package name
// this is simply emitted as a hierarchical label e.g. 'api.grpc'.
const namedLogger = "candlesV2"

// Config represent the configuration of the candle v2 package
type Config struct {
	Level         encoding.LogLevel   `long:"log-level"`
	CandleStore   CandleStoreConfig   `group:"CandleStore" namespace:"candlestore"`
	CandleUpdates CandleUpdatesConfig `group:"CandleUpdates" namespace:"candleupdates"`
}

type CandleStoreConfig struct {
	DefaultCandleIntervals string `string:"default-candle-intervals" description:"candles with the given intervals will always be created and exist by default"`
}

type CandleUpdatesConfig struct {
	CandleUpdatesStreamBufferSize int               `long:"candle-updates-stream-buffer-size" description:"buffer size used by the candle events stream for the per client per candle channel"`
	CandleUpdatesStreamInterval   encoding.Duration `long:"candle-updates-stream-interval" description:"The time between sending updated candles"`
	CandlesFetchTimeout           encoding.Duration `long:"candles-fetch-timeout" description:"Maximum time permissible to fetch candles"`
}

// NewDefaultConfig creates an instance of the package specific configuration, given a
// pointer to a logger instance to be used for logging within the package.
func NewDefaultConfig() Config {
	return Config{
		Level: encoding.LogLevel{Level: logging.InfoLevel},
		CandleStore: CandleStoreConfig{
			DefaultCandleIntervals: "1 minute,5 minutes,15 minutes,1 hour,6 hours,1 day,1 week,1 month,3 months,1 year",
		},
		CandleUpdates: CandleUpdatesConfig{
			CandleUpdatesStreamBufferSize: 100,
			CandleUpdatesStreamInterval:   encoding.Duration{Duration: time.Second},
			CandlesFetchTimeout:           encoding.Duration{Duration: 10 * time.Second},
		},
	}
}
