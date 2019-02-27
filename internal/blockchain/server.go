package blockchain

import (
	"fmt"

	"vega/internal/execution"
	"vega/internal/logging"
	"vega/internal/vegatime"

	"github.com/tendermint/tendermint/abci/server"
	cmn "github.com/tendermint/tmlibs/common"
)

type Server struct {
	*Config
	abci      *AbciApplication
	execution execution.Engine
	time      vegatime.Service
}

// NewServer creates a new instance of the the blockchain server given configuration,
// stats provider, time service and execution engine.
func NewServer(config *Config, stats *Stats, ex execution.Engine, time vegatime.Service) *Server {
	app := NewAbciApplication(config, stats, ex, time)
	return &Server{config, app, ex, time}
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
	
	// Wait forever
	cmn.TrapSignal(func() {
		srv.Stop()
	})
	return nil
}
