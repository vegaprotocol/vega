// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package admin

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"

	"code.vegaprotocol.io/vega/core/nodewallets"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/paths"

	"github.com/gorilla/mux"
	"github.com/gorilla/rpc"
	"github.com/gorilla/rpc/json"
)

type ProtocolUpgradeService interface {
	// is vega core ready to be stopped and upgraded
	GetUpgradeStatus() types.UpgradeStatus
}

// Server implement a socket server allowing to run simple RPC commands.
type Server struct {
	log *logging.Logger
	cfg Config
	srv *http.Server

	nodeWallet             *NodeWallet
	protocolUpgradeService *ProtocolUpgradeAdminService
}

// NewNonValidatorServer returns a new instance of the non-validator RPC socket server.
func NewNonValidatorServer(
	log *logging.Logger,
	config Config,
	protocolUpgradeService ProtocolUpgradeService,
) (*Server, error) {
	// setup logger
	log = log.Named(nvServerNamedLogger)
	log.SetLevel(config.Level.Get())

	return &Server{
		log:                    log,
		cfg:                    config,
		nodeWallet:             nil,
		srv:                    nil,
		protocolUpgradeService: NewProtocolUpgradeService(protocolUpgradeService),
	}, nil
}

// NewValidatorServer returns a new instance of the validator RPC socket server.
func NewValidatorServer(
	log *logging.Logger,
	config Config,
	vegaPaths paths.Paths,
	nodeWalletPassphrase string,
	nodeWallets *nodewallets.NodeWallets,
	protocolUpgradeService ProtocolUpgradeService,
) (*Server, error) {
	// setup logger
	log = log.Named(vServerNamedLogger)
	log.SetLevel(config.Level.Get())

	nodeWallet, err := NewNodeWallet(log, vegaPaths, nodeWalletPassphrase, nodeWallets)
	if err != nil {
		return nil, fmt.Errorf("failed to create node wallet service: %w", err)
	}

	return &Server{
		log:                    log,
		cfg:                    config,
		nodeWallet:             nodeWallet,
		srv:                    nil,
		protocolUpgradeService: NewProtocolUpgradeService(protocolUpgradeService),
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

	// TODO(): not updating the actual server for now, may need to look at this later
	// e.g restart the http server on another port or whatever
	s.cfg = cfg
}

// Start starts the server.
func (s *Server) Start() {
	logger := s.log

	logger.Info("Starting Server<>RPC based API",
		logging.String("socket-path", s.cfg.Server.SocketPath),
		logging.String("http-path", s.cfg.Server.HTTPPath))

	rs := rpc.NewServer()
	rs.RegisterCodec(json.NewCodec(), "application/json")
	rs.RegisterCodec(json.NewCodec(), "application/json;charset=UTF-8")

	if s.nodeWallet != nil {
		if err := rs.RegisterService(s.nodeWallet, ""); err != nil {
			logger.Panic("Failed to register node wallet service", logging.Error(err))
		}
	}
	if err := rs.RegisterService(s.protocolUpgradeService, "protocolupgrade"); err != nil {
		logger.Panic("Failed to register protocol upgrade service", logging.Error(err))
	}

	r := mux.NewRouter()
	r.Handle(s.cfg.Server.HTTPPath, rs)

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
	if err := s.srv.Serve(l); err != nil && err != http.ErrServerClosed {
		logger.Error("Error serving admin API", logging.Error(err))
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
