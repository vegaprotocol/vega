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

package rest

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strconv"

	"code.vegaprotocol.io/vega/datanode/gateway"
	"code.vegaprotocol.io/vega/logging"
	protoapi "code.vegaprotocol.io/vega/protos/data-node/api/v1"
	protoapiv2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	vegaprotoapi "code.vegaprotocol.io/vega/protos/vega/api/v1"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/rs/cors"
	"github.com/tmc/grpc-websocket-proxy/wsproxy"
	"go.elastic.co/apm/module/apmhttp"
	"google.golang.org/grpc"
)

const (
	namedLogger = "gateway.restproxy"
)

// ProxyServer implement a rest server acting as a proxy to the grpc api.
type ProxyServer struct {
	log *logging.Logger
	gateway.Config
	srv *http.Server
}

// NewProxyServer returns a new instance of the rest proxy server.
func NewProxyServer(log *logging.Logger, config gateway.Config) *ProxyServer {
	// setup logger
	log = log.Named(namedLogger)
	log.SetLevel(config.Level.Get())

	return &ProxyServer{
		log:    log,
		Config: config,
		srv:    nil,
	}
}

// ReloadConf update the internal configuration of the server.
func (s *ProxyServer) ReloadConf(cfg gateway.Config) {
	s.log.Info("reloading confioguration")
	if s.log.GetLevel() != cfg.Level.Get() {
		s.log.Info("updating log level",
			logging.String("old", s.log.GetLevel().String()),
			logging.String("new", cfg.Level.String()),
		)
		s.log.SetLevel(cfg.Level.Get())
	}

	// TODO(): not updating the the actual server for now, may need to look at this later
	// e.g restart the http server on another port or whatever
	s.Config = cfg
}

// Start start the server.
func (s *ProxyServer) Start() error {
	logger := s.log

	logger.Info("Starting REST<>GRPC based API",
		logging.String("addr", s.REST.IP),
		logging.Int("port", s.REST.Port))

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	restAddr := net.JoinHostPort(s.REST.IP, strconv.Itoa(s.REST.Port))
	grpcAddr := net.JoinHostPort(s.Node.IP, strconv.Itoa(s.Node.Port))
	jsonPB := &JSONPb{
		EmitDefaults: true,
		Indent:       "  ", // formatted json output
		OrigName:     false,
	}

	mux := runtime.NewServeMux(
		runtime.WithMarshalerOption(runtime.MIMEWildcard, jsonPB),
	)

	opts := []grpc.DialOption{grpc.WithInsecure()}
	if err := vegaprotoapi.RegisterCoreServiceHandlerFromEndpoint(ctx, mux, grpcAddr, opts); err != nil {
		logger.Panic("Failure registering trading handler for REST proxy endpoints", logging.Error(err))
	}
	if err := protoapi.RegisterTradingDataServiceHandlerFromEndpoint(ctx, mux, grpcAddr, opts); err != nil {
		logger.Panic("Failure registering trading handler for REST proxy endpoints", logging.Error(err))
	}
	if err := protoapiv2.RegisterTradingDataServiceHandlerFromEndpoint(ctx, mux, grpcAddr, opts); err != nil {
		logger.Panic("Failure registering trading handler for REST proxy endpoints", logging.Error(err))
	}

	// CORS support
	handler := cors.Default().Handler(mux)
	handler = healthCheckMiddleware(handler)
	handler = gateway.RemoteAddrMiddleware(logger, handler)
	// Gzip encoding support
	handler = newGzipHandler(*logger, handler.(http.HandlerFunc))
	// Metric support
	handler = gateway.MetricCollectionMiddleware(handler)
	handler = wsproxy.WebsocketProxy(handler)

	// APM
	if s.REST.APMEnabled {
		handler = apmhttp.Wrap(handler)
	}

	s.srv = &http.Server{
		Addr:    restAddr,
		Handler: handler,
	}

	// Start http server on port specified
	err := s.srv.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failure serving REST proxy API %w", err)
	}

	return nil
}

// Stop stops the server.
func (s *ProxyServer) Stop() {
	if s.srv != nil {
		s.log.Info("Stopping REST<>GRPC based API")

		if err := s.srv.Shutdown(context.Background()); err != nil {
			s.log.Error("Failed to stop REST<>GRPC based API cleanly",
				logging.Error(err))
		}
	}
}

func healthCheckMiddleware(f http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.Write([]byte("ok"))
			w.WriteHeader(http.StatusOK)
			return
		}
		f.ServeHTTP(w, r)
	}
}
