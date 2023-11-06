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
	"context"
	"net"
	"net/http"
	"time"

	libhttp "code.vegaprotocol.io/vega/libs/http"
	"code.vegaprotocol.io/vega/logging"

	"github.com/rs/cors"
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
