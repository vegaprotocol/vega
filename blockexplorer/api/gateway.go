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
	"net"
	"net/http"

	"github.com/rs/cors"

	libhttp "code.vegaprotocol.io/vega/libs/http"
	"code.vegaprotocol.io/vega/logging"
)

type Gateway struct {
	GatewayConfig
	httpServerMux *http.ServeMux
	log           *logging.Logger
}

type GatewayHandler interface {
	http.Handler
	Name() string
}

func NewGateway(log *logging.Logger, config GatewayConfig) *Gateway {
	log = log.Named(gatewayNamedLogger)
	return &Gateway{
		GatewayConfig: config,
		httpServerMux: http.NewServeMux(),
		log:           log,
	}
}

func (s *Gateway) Register(handler GatewayHandler, endpoint string) {
	s.log.Info("registered with api gateway", logging.String("endpoint", endpoint), logging.String("handler", handler.Name()))
	s.httpServerMux.Handle(endpoint+"/", http.StripPrefix(endpoint, handler))
}

func (s *Gateway) Serve(lis net.Listener) error {
	logAddr := logging.String("address", lis.Addr().String())
	s.log.Info("gateway starting", logAddr)
	defer s.log.Info("gateway stopping", logAddr)
	corsOptions := libhttp.CORSOptions(s.CORS)
	handler := cors.New(corsOptions).Handler(s.httpServerMux)
	return http.Serve(lis, handler)
}
