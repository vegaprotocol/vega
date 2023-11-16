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

package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"

	"code.vegaprotocol.io/vega/datanode/gateway"
	gql "code.vegaprotocol.io/vega/datanode/gateway/graphql"
	"code.vegaprotocol.io/vega/datanode/gateway/rest"
	libhttp "code.vegaprotocol.io/vega/libs/http"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/paths"

	"github.com/rs/cors"
	"golang.org/x/sync/errgroup"
)

type Server struct {
	cfg       *gateway.Config
	log       *logging.Logger
	vegaPaths paths.Paths

	rest *rest.ProxyServer
	gql  *gql.GraphServer

	srv *http.Server
}

const namedLogger = "gateway"

func New(cfg gateway.Config, log *logging.Logger, vegaPaths paths.Paths) *Server {
	log = log.Named(namedLogger)
	log.SetLevel(cfg.Level.Get())

	return &Server{
		log:       log,
		cfg:       &cfg,
		vegaPaths: vegaPaths,
	}
}

func (srv *Server) Start(ctx context.Context) error {
	eg, ctx := errgroup.WithContext(ctx)

	// <--- cors support - configure for production
	corsOptions := libhttp.CORSOptions(srv.cfg.CORS)
	corz := cors.New(corsOptions)
	// cors support - configure for production --->

	var gqlHandler, restHandler http.Handler
	if srv.cfg.GraphQL.Enabled {
		var err error
		srv.gql, err = gql.New(srv.log, *srv.cfg, srv.vegaPaths)
		if err != nil {
			return err
		}
		gqlHandler, err = srv.gql.Start()
		if err != nil {
			return err
		}
	}

	if srv.cfg.REST.Enabled {
		srv.rest = rest.NewProxyServer(srv.log, *srv.cfg, srv.vegaPaths)

		var err error
		restHandler, err = srv.rest.Start(ctx)
		if err != nil {
			return err
		}
	}

	handlr := corz.Handler(
		&Handler{
			gqlPrefix:   srv.cfg.GraphQL.Endpoint,
			gqlHandler:  gqlHandler,
			restHandler: restHandler,
		},
	)

	port := srv.cfg.Port
	ip := srv.cfg.IP

	srv.log.Info("Starting http based API", logging.String("addr", ip), logging.Int("port", port))

	addr := net.JoinHostPort(ip, strconv.Itoa(port))

	tlsConfig, fallback, err := gateway.GenerateTlsConfig(srv.cfg, srv.vegaPaths)
	if err != nil {
		return fmt.Errorf("problem with HTTPS configuration: %w", err)
	}
	srv.srv = &http.Server{
		Addr:      addr,
		Handler:   handlr,
		TLSConfig: tlsConfig,
	}

	var fallbacksrv *http.Server
	if srv.cfg.REST.Enabled || srv.cfg.GraphQL.Enabled {
		eg.Go(func() error {
			if srv.srv.TLSConfig != nil {
				if fallback != nil {
					eg.Go(func() error {
						fallbacksrv = &http.Server{Addr: ":http", Handler: fallback}
						// serve HTTP, which will redirect automatically to HTTPS
						err := fallbacksrv.ListenAndServe()
						if err != nil && err != http.ErrServerClosed {
							return fmt.Errorf("failed start fallback http server: %w", err)
						}
						return nil
					})
				}
				err = srv.srv.ListenAndServeTLS("", "")
			} else {
				srv.log.Warn("GraphQL server is not configured to use HTTPS, which is required for subscriptions to work. Please see README.md for help configuring")
				err = srv.srv.ListenAndServe()
			}
			if err != nil && err != http.ErrServerClosed {
				return fmt.Errorf("failed to listen and serve on graphQL server: %w", err)
			}

			return nil
		})

		eg.Go(func() error {
			<-ctx.Done()
			srv.stop()
			if fallbacksrv != nil {
				fallbacksrv.Shutdown(context.Background())
			}
			return nil
		})
	}

	return eg.Wait()
}

// stop stops the server.
func (srv *Server) stop() {
	if srv.srv != nil {
		srv.log.Info("stopping http based API")

		if err := srv.srv.Shutdown(context.Background()); err != nil {
			srv.log.Error("Failed to stop http based API cleanly",
				logging.Error(err))
		}
	}
}

type Handler struct {
	gqlPrefix   string
	restHandler http.Handler
	gqlHandler  http.Handler
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, h.gqlPrefix) {
		if h.gqlHandler != nil {
			h.gqlHandler.ServeHTTP(w, r)
			return
		}
	} else if h.restHandler != nil {
		h.restHandler.ServeHTTP(w, r)
		return
	}

	// cover for unknow routes, or disabled servers
	http.NotFound(w, r)
}
