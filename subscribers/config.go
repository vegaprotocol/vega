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

package subscribers

import (
	"code.vegaprotocol.io/data-node/config/encoding"
	"code.vegaprotocol.io/data-node/logging"
)

const (
	namedLogger = "subscribers"
)

// Config represent the configuration of the subscribers package
type Config struct {
	OrderEventLogLevel  encoding.LogLevel `long:"order-event-log-level"`
	MarketEventLogLevel encoding.LogLevel `long:"market-even-log-level"`
}

// NewDefaultConfig creates an instance of the package specific configuration, given a
// pointer to a logger instance to be used for logging within the package.
func NewDefaultConfig() Config {
	return Config{
		MarketEventLogLevel: encoding.LogLevel{Level: logging.InfoLevel},
		OrderEventLogLevel:  encoding.LogLevel{Level: logging.InfoLevel},
	}
}
