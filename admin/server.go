package admin

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"

	"code.vegaprotocol.io/shared/paths"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/nodewallets"
	"github.com/gorilla/mux"
	"github.com/gorilla/rpc"
	"github.com/gorilla/rpc/json"
)

// Server implement a socket server allowing to run simple RPC commands.
type Server struct {
	log *logging.Logger
	cfg Config
	srv *http.Server

	nodeWallet *NodeWallet
}

// NewServer returns a new instance of the RPC socket server.
func NewServer(
	log *logging.Logger,
	config Config,
	vegaPaths paths.Paths,
	nodeWalletPassphrase string,
	nodeWallets *nodewallets.NodeWallets,
) (*Server, error) {
	// setup logger
	log = log.Named(namedLogger)
	log.SetLevel(config.Level.Get())

	nodeWallet, err := NewNodeWallet(log, vegaPaths, nodeWalletPassphrase, nodeWallets)
	if err != nil {
		return nil, fmt.Errorf("failed to create node wallet service: %w", err)
	}

	return &Server{
		log:        log,
		cfg:        config,
		nodeWallet: nodeWallet,
		srv:        nil,
	}, nil
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

	// TODO(): not updating the the actual server for now, may need to look at this later
	// e.g restart the http server on another port or whatever
	s.cfg = cfg
}

// Start start the server.
func (s *Server) Start() {
	logger := s.log

	logger.Info("Starting Server<>RPC based API",
		logging.String("socket-path", s.cfg.Server.SocketPath),
		logging.String("http-path", s.cfg.Server.HttpPath))

	rs := rpc.NewServer()
	rs.RegisterCodec(json.NewCodec(), "application/json")
	rs.RegisterCodec(json.NewCodec(), "application/json;charset=UTF-8")

	rs.RegisterService(s.nodeWallet, "")
	r := mux.NewRouter()
	r.Handle(s.cfg.Server.HttpPath, rs)

	// Try to remove just in case
	os.Remove(s.cfg.Server.SocketPath)

	l, err := net.Listen("unix", s.cfg.Server.SocketPath)
	if err != nil {
		logger.Panic("Failed to open unix socket", logging.Error(err))
	}

	s.srv = &http.Server{
		Handler: r,
	}

	logger.Info("Serving Server<>RPC based API")
	if err := s.srv.Serve(l); err != nil {
		logger.Panic("Failed to serve Server<>RPC based API", logging.Error(err))
	}
}

// Stop stops the server.
func (s *Server) Stop() {
	if s.srv != nil {
		s.log.Info("Stopping Server<>RPC based API")

		if err := s.srv.Shutdown(context.Background()); err != nil {
			s.log.Error("Failed to stop Server<>RPC based API cleanly",
				logging.Error(err))
		}
	}
}
