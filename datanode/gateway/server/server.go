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

package server

import (
	"context"

	"golang.org/x/sync/errgroup"

	"code.vegaprotocol.io/vega/datanode/gateway"
	gql "code.vegaprotocol.io/vega/datanode/gateway/graphql"
	"code.vegaprotocol.io/vega/datanode/gateway/rest"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/paths"
)

type Server struct {
	cfg       *gateway.Config
	log       *logging.Logger
	vegaPaths paths.Paths

	rest *rest.ProxyServer
	gql  *gql.GraphServer
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

	if srv.cfg.GraphQL.Enabled {
		var err error
		srv.gql, err = gql.New(srv.log, *srv.cfg, srv.vegaPaths)
		if err != nil {
			return err
		}
		eg.Go(func() error { return srv.gql.Start() })
	}

	if srv.cfg.REST.Enabled {
		srv.rest = rest.NewProxyServer(srv.log, *srv.cfg, srv.vegaPaths)
		eg.Go(func() error { return srv.rest.Start() })
	}

	if srv.cfg.REST.Enabled || srv.cfg.GraphQL.Enabled {
		eg.Go(func() error {
			<-ctx.Done()
			srv.stop()
			return nil
		})
	}

	return eg.Wait()
}

func (srv *Server) stop() {
	if s := srv.rest; s != nil {
		s.Stop()
	}

	if s := srv.gql; s != nil {
		s.Stop()
	}
}
