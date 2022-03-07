package server

import (
	"context"

	"code.vegaprotocol.io/data-node/gateway"
	gql "code.vegaprotocol.io/data-node/gateway/graphql"
	"code.vegaprotocol.io/data-node/gateway/rest"
	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/shared/paths"
	"golang.org/x/sync/errgroup"
)

type Server struct {
	cfg       *gateway.Config
	log       *logging.Logger
	vegaPaths paths.Paths

	rest *rest.ProxyServer
	gql  *gql.GraphServer
}

func New(cfg gateway.Config, log *logging.Logger, vegaPaths paths.Paths) *Server {
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
		srv.rest = rest.NewProxyServer(srv.log, *srv.cfg)
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
