package api

import (
	"context"
	"fmt"
	"net"

	"code.vegaprotocol.io/vega/logging"

	"google.golang.org/grpc"
)

type Server struct {
	Config

	log *logging.Logger
	srv *grpc.Server

	ctx   context.Context
	cfunc context.CancelFunc
}

func New(log *logging.Logger, cfg Config) *Server {
	log = log.Named(namedLogger)
	log.SetLevel(cfg.Level.Get())
	ctx, cfunc := context.WithCancel(context.Background())

	intercept := grpc.UnaryInterceptor(remoteAddrInterceptor(log))
	srv := grpc.NewServer(intercept)

	return &Server{
		Config: cfg,
		log:    log,
		ctx:    ctx,
		cfunc:  cfunc,
		srv:    srv,
	}
}

func (s *Server) GRPC() *grpc.Server {
	return s.srv
}

func (s *Server) Start() error {
	s.log.Info("Starting gRPC based API",
		logging.String("addr", s.IP),
		logging.Int("port", s.Port))

	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", s.IP, s.Port))
	if err != nil {
		s.log.Error("Failure listening on gRPC port",
			logging.Int("port", s.Port),
			logging.Error(err))
		return err
	}

	err = s.srv.Serve(lis)
	if err != nil {
		s.log.Error("Failure serving gRPC API",
			logging.Error(err))
		return err
	}
	return nil
}

// Stop stops the GRPC server
func (g *Server) Stop() {
	if g.srv != nil {
		g.log.Info("Stopping gRPC based API")
		g.cfunc()
		g.srv.GracefulStop()
	}
}
