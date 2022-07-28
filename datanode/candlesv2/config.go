// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package candlesv2

import (
	"time"

	"code.vegaprotocol.io/data-node/datanode/config/encoding"
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
		CandleUpdates: CandleUpdatesConfig{
			CandleUpdatesStreamBufferSize: 100,
			CandleUpdatesStreamInterval:   encoding.Duration{Duration: time.Second},
			CandlesFetchTimeout:           encoding.Duration{Duration: 10 * time.Second},
		},
	}
}
