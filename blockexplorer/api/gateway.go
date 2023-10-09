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
	"context"
	"net"
	"net/http"
	"time"

	"github.com/rs/cors"

	libhttp "code.vegaprotocol.io/vega/libs/http"
	"code.vegaprotocol.io/vega/logging"
)

type Gateway struct {
	GatewayConfig
	httpServerMux *http.ServeMux
	log           *logging.Logger
	srv           *http.Server
	lis           net.Listener
}

type GatewayHandler interface {
	http.Handler
	Name() string
}

func NewGateway(log *logging.Logger, config GatewayConfig, lis net.Listener) *Gateway {
	log = log.Named(gatewayNamedLogger)
	return &Gateway{
		GatewayConfig: config,
		httpServerMux: http.NewServeMux(),
		log:           log,
		lis:           lis,
	}
}

func (s *Gateway) Register(handler GatewayHandler, endpoint string) {
	s.log.Info("Registering with API gateway", logging.String("endpoint", endpoint), logging.String("handler", handler.Name()))
	s.httpServerMux.Handle(endpoint+"/", http.StripPrefix(endpoint, handler))
}

func (s *Gateway) Serve() error {
	s.log.Info("Starting gateway", logging.String("address", s.lis.Addr().String()))
	corsOptions := libhttp.CORSOptions(s.CORS)
	handler := cors.New(corsOptions).Handler(s.httpServerMux)
	srv := &http.Server{Handler: handler}
	s.srv = srv
	return s.srv.Serve(s.lis)
}

func (s *Gateway) Stop() {
	if s.srv != nil {
		s.log.Info("Stopping gateway", logging.String("address", s.lis.Addr().String()))
		ctxWithTimeout, cancelFunc := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancelFunc()
		_ = s.srv.Shutdown(ctxWithTimeout)
	}
}
