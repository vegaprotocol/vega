package blockchain

import (
	"errors"
	"fmt"

	"code.vegaprotocol.io/vega/internal/logging"
	"code.vegaprotocol.io/vega/internal/vegatime"

	"github.com/tendermint/tendermint/abci/server"
	cmn "github.com/tendermint/tendermint/libs/common"
)

type Server struct {
	*Config
	abci *AbciApplication
	time vegatime.Service
	srv  cmn.Service
}

func NewServer(config *Config, stats *Stats, time vegatime.Service, app *AbciApplication) *Server {
	return &Server{
		Config: config,
		abci:   app,
		time:   time,
		srv:    nil,
	}
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

func (s *Server) Stop() error {
	if s.srv != nil {
		return s.srv.Stop()
	}
	return errors.New("server not started")
}
