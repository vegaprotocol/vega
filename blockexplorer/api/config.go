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

package api

import (
	"github.com/inhies/go-bytesize"

	"code.vegaprotocol.io/vega/blockexplorer/api/grpc"
	"code.vegaprotocol.io/vega/libs/config/encoding"
	libhttp "code.vegaprotocol.io/vega/libs/http"
	"code.vegaprotocol.io/vega/logging"
)

var (
	portalNamedLogger  = "api.portal"
	gatewayNamedLogger = "api.gateway"
	restNamedLogger    = "api.rest"
	grpcUINamedLogger  = "api.grpc-ui"
)

type Config struct {
	GRPC          grpc.Config   `group:"grpc api" namespace:"grpc"`
	GRPCUI        GRPCUIConfig  `group:"grpc web ui" namespace:"grpcui"`
	REST          RESTConfig    `group:"rest api" namespace:"rest"`
	Gateway       GatewayConfig `group:"gateway" namespace:"grpcui"`
	ListenAddress string        `long:"listen-address" description:"the IP address that our sever will listen on"`
	ListenPort    uint16        `long:"listen-port" description:"the port that our sever will listen on"`
}

func NewDefaultConfig() Config {
	return Config{
		GRPC:          grpc.NewDefaultConfig(),
		GRPCUI:        NewDefaultGRPCUIConfig(),
		REST:          NewDefaultRESTConfig(),
		Gateway:       NewDefaultGatewayConfig(),
		ListenAddress: "0.0.0.0",
		ListenPort:    1515,
	}
}

type GRPCUIConfig struct {
	Enabled        encoding.Bool     `long:"enabled"`
	Endpoint       string            `long:"endpoint"`
	Level          encoding.LogLevel `long:"log-level" choice:"debug" choice:"info" choice:"warning"`
	MaxPayloadSize encoding.ByteSize `long:"max-payload-size" description:"Maximum size of GRPC messages the UI will accept from the GRPC server (e.g. 4mb)"`
}

func NewDefaultGRPCUIConfig() GRPCUIConfig {
	return GRPCUIConfig{
		Enabled:        encoding.Bool(true),
		Endpoint:       "/grpc",
		Level:          encoding.LogLevel{Level: logging.InfoLevel},
		MaxPayloadSize: encoding.ByteSize(4 * bytesize.MB),
	}
}

type GatewayConfig struct {
	CORS libhttp.CORSConfig `long:"cors" description:"CORS allowed origins"`
}

func NewDefaultGatewayConfig() GatewayConfig {
	return GatewayConfig{
		CORS: libhttp.CORSConfig{
			AllowedOrigins: []string{"*"},
			MaxAge:         7200,
		},
	}
}

type RESTConfig struct {
	Level    encoding.LogLevel `long:"log-level" choice:"debug" choice:"info" choice:"warning"`
	Enabled  encoding.Bool     `long:"enabled"`
	Endpoint string            `long:"endpoint"`
}

func NewDefaultRESTConfig() RESTConfig {
	return RESTConfig{
		Level:    encoding.LogLevel{Level: logging.InfoLevel},
		Enabled:  encoding.Bool(true),
		Endpoint: "/rest",
	}
}
