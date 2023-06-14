// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.DATANODE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

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
	Port int    `description:"Listen for connection on port <port>" long:"port"`
	IP   string `description:"Bind to address <ip>"                 long:"ip"`
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
			IP:   "0.0.0.0",
			Port: 3008,
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
			IP:   "0.0.0.0",
			Port: 3007,
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
