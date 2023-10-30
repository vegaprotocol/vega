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

	"code.vegaprotocol.io/vega/libs/config/encoding"
	libhttp "code.vegaprotocol.io/vega/libs/http"
	"code.vegaprotocol.io/vega/logging"
)

// namedLogger is the identifier for package and should ideally match the package name
// this is simply emitted as a hierarchical label e.g. 'api.grpc'.
const namedLogger = "api.grpc"

// Config represents the configuration of the api package.
type Config struct {
	Level           encoding.LogLevel `long:"log-level"`
	Timeout         encoding.Duration `long:"timeout"`
	Port            int               `long:"port"`
	IP              string            `long:"ip"`
	StreamRetries   int               `long:"stream-retries"`
	DisableTxCommit bool              `long:"disable-tx-commit"`

	REST RESTServiceConfig `group:"REST" namespace:"rest"`
}

// RESTGatewayServiceConfig represent the configuration of the rest service.
type RESTServiceConfig struct {
	Port       int                `description:"Listen for connection on port <port>" long:"port"`
	IP         string             `description:"Bind to address <ip>"                 long:"ip"`
	Enabled    encoding.Bool      `choice:"true"                                      description:"Start the REST gateway" long:"enabled"`
	APMEnabled encoding.Bool      `choice:"true"                                      description:" "                      long:"apm-enabled"`
	CORS       libhttp.CORSConfig `group:"CORS"                                       namespace:"cors"`
}

// NewDefaultConfig creates an instance of the package specific configuration, given a
// pointer to a logger instance to be used for logging within the package.
func NewDefaultConfig() Config {
	return Config{
		Level:   encoding.LogLevel{Level: logging.InfoLevel},
		Timeout: encoding.Duration{Duration: 5000 * time.Millisecond},

		IP:              "0.0.0.0",
		Port:            3002,
		StreamRetries:   3,
		DisableTxCommit: true,
		REST: RESTServiceConfig{
			IP:         "0.0.0.0",
			Port:       3003,
			Enabled:    true,
			APMEnabled: true,
			CORS: libhttp.CORSConfig{
				AllowedOrigins: []string{"*"},
				MaxAge:         7200,
			},
		},
	}
}
