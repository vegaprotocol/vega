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

package eth

import (
	"time"

	"code.vegaprotocol.io/vega/libs/config/encoding"
	"code.vegaprotocol.io/vega/logging"
)

const namedLogger = "ethClient"

type Config struct {
	Level       encoding.LogLevel `long:"log-level"`
	RPCEndpoint string
	RetryDelay  encoding.Duration
	L2Configs   []L2Config
}

type L2Config struct {
	NetworkID   string
	RPCEndpoint string
}

// NewDefaultConfig creates an instance of the package specific configuration,
// given a pointer to a logger instance to be used for logging within the
// package.
func NewDefaultConfig() Config {
	return Config{
		Level: encoding.LogLevel{Level: logging.InfoLevel},
		// default to a mainnet block time duration
		RetryDelay: encoding.Duration{Duration: 15 * time.Second},
	}
}
