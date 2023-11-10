// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package candlesv2

import (
	"time"

	"code.vegaprotocol.io/vega/datanode/config/encoding"
	"code.vegaprotocol.io/vega/logging"
)

// namedLogger is the identifier for package and should ideally match the package name
// this is simply emitted as a hierarchical label e.g. 'api.grpc'.
const namedLogger = "candlesV2"

// Config represent the configuration of the candle v2 package.
type Config struct {
	Level         encoding.LogLevel   `long:"log-level"`
	CandleStore   CandleStoreConfig   `group:"CandleStore"   namespace:"candlestore"`
	CandleUpdates CandleUpdatesConfig `group:"CandleUpdates" namespace:"candleupdates"`
}

type CandleStoreConfig struct {
	DefaultCandleIntervals string `description:"candles with the given intervals will always be created and exist by default" string:"default-candle-intervals"`
}

type CandleUpdatesConfig struct {
	CandleUpdatesStreamBufferSize                int               `description:"buffer size used by the candle events stream for the per client per candle channel" long:"candle-updates-stream-buffer-size"`
	CandleUpdatesStreamInterval                  encoding.Duration `description:"The time between sending updated candles"                                           long:"candle-updates-stream-interval"`
	CandlesFetchTimeout                          encoding.Duration `description:"Maximum time permissible to fetch candles"                                          long:"candles-fetch-timeout"`
	CandleUpdatesStreamSubscriptionMsgBufferSize int               `description:"size of the buffer used to hold pending subcribe/unsubscribe requests"              long:"candle-updates-stream-subscription-buffer-size"`
}

// NewDefaultConfig creates an instance of the package specific configuration, given a
// pointer to a logger instance to be used for logging within the package.
func NewDefaultConfig() Config {
	return Config{
		Level: encoding.LogLevel{Level: logging.InfoLevel},
		CandleUpdates: CandleUpdatesConfig{
			CandleUpdatesStreamBufferSize:                100,
			CandleUpdatesStreamInterval:                  encoding.Duration{Duration: time.Second},
			CandlesFetchTimeout:                          encoding.Duration{Duration: 10 * time.Second},
			CandleUpdatesStreamSubscriptionMsgBufferSize: 100,
		},
	}
}
