package tm

import (
	"fmt"

	"code.vegaprotocol.io/vega/logging"

	"github.com/tendermint/tendermint/abci/server"
	cmn "github.com/tendermint/tendermint/libs/common"
)

// Server is an abstraction over the abci server
type Server struct {
	Config
	log  *logging.Logger
	abci *AbciApplication
	srv  cmn.Service
}

// NewServer instantiate a new server
func NewServer(log *logging.Logger, config Config, app *AbciApplication) *Server {
	// setup logger
	log = log.Named(namedLogger)
	log.SetLevel(config.Level.Get())

	return &Server{
		log:    log,
		Config: config,
		abci:   app,
		srv:    nil,
	}
}

// ReloadConf update the internal configuration
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
	s.Config = cfg
}

// Start configures and runs a new socket based ABCI tendermint blockchain
// server for the VEGA application.
func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.ServerAddr, s.ServerPort)
	srv, err := server.NewServer(addr, "socket", s.abci)
	if err != nil {
		return err
	}

	s.log.Info("Starting abci-blockchain socket server",
		logging.String("addr", s.ServerAddr),
		logging.Int("port", s.ServerPort))

	if err := srv.Start(); err != nil {
		return err
	}

	s.srv = srv

	return nil
}

// Stop the abci server
func (s *Server) Stop() {
	if s.srv != nil {
		s.log.Info("Stopping abci-blockchain socket server")
		if err := s.srv.Stop(); err != nil {
			s.log.Error("Failed to stop abci-blockchain socket server cleanly",
				logging.Error(err))
		}
	}
}
