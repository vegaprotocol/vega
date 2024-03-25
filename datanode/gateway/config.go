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

//lint:file-ignore SA5008 duplicated struct tags are ok for config

package gateway

import (
	"time"

	"code.vegaprotocol.io/vega/datanode/config/encoding"
	"code.vegaprotocol.io/vega/datanode/ratelimit"
	libhttp "code.vegaprotocol.io/vega/libs/http"
	"code.vegaprotocol.io/vega/logging"
)

// ServerConfig represent the configuration of a server in vega.
type ServerConfig struct {
	Port       int    `description:"Listen for connection on port <port>" long:"port"`
	IP         string `description:"Bind to address <ip>"                 long:"ip"`
	MaxMsgSize int    `description:"Max message size in bytes"            long:"max-msg-size"`
}

// GraphqlServiceConfig represents the configuration of the gateway.
type GraphqlServiceConfig struct {
	Enabled         encoding.Bool `description:"Start the GraphQl gateway"             long:"enabled"`
	ComplexityLimit int           `long:"complexity-limit"`
	Endpoint        string        `description:"Endpoint to expose the graphql API at" long:"endpoint"`
}

// RESTGatewayServiceConfig represent the configuration of the rest service.
type RESTGatewayServiceConfig struct {
	Enabled    encoding.Bool `choice:"true" choice:"false" description:"Start the REST gateway" long:"enabled"`
	APMEnabled encoding.Bool `choice:"true" choice:"false" description:" "                      long:"apm-enabled"`
}

// Config represents the general configuration for the gateway.
type Config struct {
	ServerConfig
	Level                    encoding.LogLevel        `choice:"debug"                                                                  choice:"info"                      choice:"warning" long:"log-level"`
	Timeout                  encoding.Duration        `long:"timeout"`
	Node                     ServerConfig             `group:"Node"                                                                    namespace:"node"`
	GraphQL                  GraphqlServiceConfig     `group:"GraphQL"                                                                 namespace:"graphql"`
	REST                     RESTGatewayServiceConfig `group:"REST"                                                                    namespace:"rest"`
	SubscriptionRetries      int                      `description:" "                                                                 long:"subscription-retries"`
	GraphQLPlaygroundEnabled encoding.Bool            `description:"Enables the GraphQL playground"                                    long:"graphql-playground"`
	MaxSubscriptionPerClient uint32                   `description:"Maximum graphql subscriptions allowed per client"                  long:"max-subscription-per-client"`
	CORS                     libhttp.CORSConfig       `group:"CORS"                                                                    namespace:"cors"`
	HTTPSEnabled             encoding.Bool            `description:"If true, GraphQL gateway will require an HTTPS connection"         long:"https-enabled"`
	AutoCertDomain           string                   `description:"Automatically generate and sign https certificate via LetsEncrypt" long:"auto-cert-domain"`
	CertificateFile          string                   `description:"Path to SSL certificate, if using HTTPS but not autocert"          long:"certificate-file"`
	KeyFile                  string                   `description:"Path to private key, if using HTTPS but not autocert"              long:"key-file"`
	RateLimit                ratelimit.Config         `group:"RateLimits"                                                              namespace:"rate-limits"`
}

// NewDefaultConfig creates an instance of the package specific configuration, given a
// pointer to a logger instance to be used for logging within the package.
func NewDefaultConfig() Config {
	return Config{
		ServerConfig: ServerConfig{
			IP:         "0.0.0.0",
			Port:       3008,
			MaxMsgSize: 20 * 1024 * 1024,
		},
		Level:        encoding.LogLevel{Level: logging.InfoLevel},
		Timeout:      encoding.Duration{Duration: 5 * time.Second},
		HTTPSEnabled: false,
		GraphQL: GraphqlServiceConfig{
			Enabled:  true,
			Endpoint: "/graphql",
		},
		REST: RESTGatewayServiceConfig{
			Enabled:    true,
			APMEnabled: true,
		},
		Node: ServerConfig{
			IP:         "0.0.0.0",
			Port:       3007,
			MaxMsgSize: 20 * 1024 * 1024,
		},
		SubscriptionRetries:      3,
		GraphQLPlaygroundEnabled: true,
		MaxSubscriptionPerClient: 250,
		CORS: libhttp.CORSConfig{
			AllowedOrigins: []string{"*"},
			MaxAge:         7200,
		},
		RateLimit: ratelimit.NewDefaultConfig(),
	}
}
