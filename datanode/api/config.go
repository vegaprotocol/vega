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

package api

import (
	"time"

	"code.vegaprotocol.io/vega/datanode/config/encoding"
	"code.vegaprotocol.io/vega/datanode/ratelimit"
	"code.vegaprotocol.io/vega/logging"
)

// namedLogger is the identifier for package and should ideally match the package name
// this is simply emitted as a hierarchical label e.g. 'api.grpc'.
const namedLogger = "api.grpc"

// Config represents the configuration of the api package.
type Config struct {
	Level                    encoding.LogLevel `long:"log-level"`
	Timeout                  encoding.Duration `long:"timeout"`
	Port                     int               `long:"port"`
	WebUIPort                int               `long:"web-ui-port"`
	WebUIEnabled             encoding.Bool     `long:"web-ui-enabled"`
	Reflection               encoding.Bool     `long:"reflection"`
	IP                       string            `long:"ip"`
	StreamRetries            int               `long:"stream-retries"`
	CoreNodeIP               string            `long:"core-node-ip"`
	CoreNodeGRPCPort         int               `long:"core-node-grpc-port"`
	RateLimit                ratelimit.Config  `group:"rate-limits"`
	MaxSubscriptionPerClient uint32            `long:"max-subscription-per-client"`
	MaxMsgSize               int               `long:"max-msg-size"`
}

// NewDefaultConfig creates an instance of the package specific configuration, given a
// pointer to a logger instance to be used for logging within the package.
func NewDefaultConfig() Config {
	return Config{
		Level:   encoding.LogLevel{Level: logging.InfoLevel},
		Timeout: encoding.Duration{Duration: 5000 * time.Millisecond},

		IP:                       "0.0.0.0",
		Port:                     3007,
		WebUIPort:                3006,
		WebUIEnabled:             false,
		Reflection:               false,
		StreamRetries:            3,
		CoreNodeIP:               "127.0.0.1",
		CoreNodeGRPCPort:         3002,
		RateLimit:                ratelimit.NewDefaultConfig(),
		MaxSubscriptionPerClient: 250,
		MaxMsgSize:               20 * 1024 * 1024,
	}
}
