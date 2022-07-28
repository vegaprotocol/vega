// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package matching

import (
	"code.vegaprotocol.io/vega/core/config/encoding"
	"code.vegaprotocol.io/vega/logging"
)

// namedLogger is the identifier for package and should ideally match the package name
// this is simply emitted as a hierarchical label e.g. 'api.grpc'.
const namedLogger = "matching"

// Config represents the configuration of the Matching engine.
type Config struct {
	Level encoding.LogLevel `long:"log-level"`

	LogPriceLevelsDebug   bool `long:"log-price-levels-debug"`
	LogRemovedOrdersDebug bool `long:"log-removed-orders-debug"`
}

// NewDefaultConfig creates an instance of the package specific configuration, given a
// pointer to a logger instance to be used for logging within the package.
func NewDefaultConfig() Config {
	return Config{
		Level:                 encoding.LogLevel{Level: logging.InfoLevel},
		LogPriceLevelsDebug:   false,
		LogRemovedOrdersDebug: false,
	}
}
