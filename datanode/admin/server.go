package admin

import (
	"context"
	"net"
	"net/http"
	"os"

	"code.vegaprotocol.io/vega/paths"

	"github.com/gorilla/mux"
	"github.com/gorilla/rpc"
	"github.com/gorilla/rpc/json"

	"code.vegaprotocol.io/vega/logging"
)

type Server struct {
	log                        *logging.Logger
	cfg                        Config
	srv                        *http.Server
	networkHistoryAdminService *NetworkHistoryAdminService
}

func NewServer(log *logging.Logger, config Config, vegaPaths paths.Paths, service *NetworkHistoryAdminService) *Server {
	// setup logger
	log = log.Named(namedLogger)
	log.SetLevel(config.Level.Get())

	return &Server{
		log:                        log,
		cfg:                        config,
		srv:                        nil,
		networkHistoryAdminService: service,
	}
}

// Start starts the RPC based API server.
func (s *Server) Start(ctx context.Context) error {
	s.log.Info("Starting Data Node Admin Server<>RPC based API",
		logging.String("socket-path", s.cfg.Server.SocketPath),
		logging.String("http-path", s.cfg.Server.HTTPPath),
	)

	rs := rpc.NewServer()
	rs.RegisterCodec(json.NewCodec(), "application/json")
	rs.RegisterCodec(json.NewCodec(), "application/json;charset=UTF-8")

	if err := rs.RegisterService(s.networkHistoryAdminService, "networkhistory"); err != nil {
		s.log.Panic("failed to register network history service", logging.Error(err))
	}

	r := mux.NewRouter()
	r.Handle(s.cfg.Server.HTTPPath, rs)

	// Try to remove the existing socket file just in case
	if err := os.Remove(s.cfg.Server.SocketPath); err != nil {
		// If we can't remove the socket and the error is not that the file doesn't exist, then we should panic
		if !os.IsNotExist(err) {
			s.log.Panic("failed to remove socket file", logging.Error(err))
		}
	}

	l, err := net.Listen("unix", s.cfg.Server.SocketPath)
	if err != nil {
		s.log.Panic("failed to open unix socket", logging.Error(err))
	}

	s.srv = &http.Server{
		Handler: r,
	}

	go func() {
		<-ctx.Done()
		s.Stop()
	}()

	s.log.Info("Data Node Admin Server<>RPC based API started")
	return s.srv.Serve(l)
}

// Stop stops the RPC based API server.
func (s *Server) Stop() {
	if s.srv != nil {
		s.log.Info("Stopping Data Node Admin Server<>RPC based API")
		if err := s.srv.Close(); err != nil {
			s.log.Error("failed to stop Data Node Admin server<>RPC based API cleanly",
				logging.Error(err))
		}
	}
}

// ReloadConf update the internal configuration of the server.
func (s *Server) ReloadConf(cfg Config) {
	s.log.Info("reloading configuration")
	if s.log.GetLevel() != cfg.Level.Get() {
		s.log.Info("updating log level",
			logging.String("old", s.log.GetLevel().String()),
			logging.String("new", cfg.Level.String()),
		)
		s.log.SetLevel(cfg.Level.Get())
	}

	// TODO(): not updating the actual server for now, may need to look at this later
	// e.g restart the http server on another port or whatever
	s.cfg = cfg
}
