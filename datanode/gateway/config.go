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
	Port int    `long:"port" description:"Listen for connection on port <port>"`
	IP   string `long:"ip" description:"Bind to address <ip>"`
}

// GraphqlServiceConfig represents the configuration of the gateway.
type GraphqlServiceConfig struct {
	Enabled         encoding.Bool `long:"enabled" description:"Start the GraphQl gateway"`
	ComplexityLimit int           `long:"complexity-limit"`
	Endpoint        string        `long:"endpoint" description:"Endpoint to expose the graphql API at"`
}

// RESTGatewayServiceConfig represent the configuration of the rest service.
type RESTGatewayServiceConfig struct {
	Enabled    encoding.Bool `long:"enabled" choice:"true" choice:"false" description:"Start the REST gateway"`
	APMEnabled encoding.Bool `long:"apm-enabled" choice:"true" choice:"false" description:" "`
}

// Config represents the general configuration for the gateway.
type Config struct {
	ServerConfig
	Level                    encoding.LogLevel        `long:"log-level" choice:"debug" choice:"info" choice:"warning"`
	Timeout                  encoding.Duration        `long:"timeout"`
	Node                     ServerConfig             `group:"Node" namespace:"node"`
	GraphQL                  GraphqlServiceConfig     `group:"GraphQL" namespace:"graphql"`
	REST                     RESTGatewayServiceConfig `group:"REST" namespace:"rest"`
	SubscriptionRetries      int                      `long:"subscription-retries" description:" "`
	GraphQLPlaygroundEnabled encoding.Bool            `long:"graphql-playground" description:"Enables the GraphQL playground"`
	MaxSubscriptionPerClient uint32                   `long:"max-subscription-per-client" description:"Maximum graphql subscriptions allowed per client"`
	CORS                     libhttp.CORSConfig       `group:"CORS" namespace:"cors"`
	HTTPSEnabled             encoding.Bool            `long:"https-enabled" description:"If true, GraphQL gateway will require an HTTPS connection"`
	AutoCertDomain           string                   `long:"auto-cert-domain" description:"Automatically generate and sign https certificate via LetsEncrypt"`
	CertificateFile          string                   `long:"certificate-file" description:"Path to SSL certificate, if using HTTPS but not autocert"`
	KeyFile                  string                   `long:"key-file" description:"Path to private key, if using HTTPS but not autocert"`
	RateLimit                ratelimit.Config         `group:"RateLimits" namespace:"rate-limits"`
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
