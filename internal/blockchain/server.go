package blockchain

import (
	"fmt"

	"vega/internal/execution"
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

func NewServer(config *Config, ex execution.Engine, time vegatime.Service) *Server {
	stats := NewStats() // package specific statistics
	app := NewAbciApplication(config, ex, time, stats)
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

	s.log.Infof("Starting abci-blockchain server socket on %s", addr)
	if err := srv.Start(); err != nil {
		return err
	}

	//vega.Statistics.Status = msg.AppStatus_CHAIN_NOT_FOUND
	// todo(cdm): app comms to get status if chain replaying etc
	// handshake stuff / security ensure app hashes match?
	//vega.SetGenesisTime(genesis.GenesisTime)
	//vega.Statistics.Status = msg.AppStatus_APP_CONNECTED

	// Wait forever
	cmn.TrapSignal(func() {
		srv.Stop()
	})
	return nil
}
