package server

import (
	"code.vegaprotocol.io/vega/gateway"
	gql "code.vegaprotocol.io/vega/gateway/graphql"
	"code.vegaprotocol.io/vega/gateway/rest"
	"code.vegaprotocol.io/vega/logging"
)

type Server struct {
	cfg *gateway.Config
	log *logging.Logger

	rest *rest.ProxyServer
	gql  *gql.GraphServer
}

func New(cfg gateway.Config, log *logging.Logger) *Server {
	return &Server{
		log: log,
		cfg: &cfg,
	}
}

func (srv *Server) Start() error {
	if srv.cfg.GraphQL.Enabled {
		var err error
		srv.gql, err = gql.New(srv.log, *srv.cfg)
		if err != nil {
			return err
		}
		go func() { srv.gql.Start() }()
	}

	if srv.cfg.REST.Enabled {
		srv.rest = rest.NewProxyServer(srv.log, *srv.cfg)
		go func() { srv.rest.Start() }()
	}

	return nil
}

func (srv *Server) Stop() {
	if s := srv.rest; s != nil {
		s.Stop()
	}

	if s := srv.gql; s != nil {
		s.Stop()
	}
}
