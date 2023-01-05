// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

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

func NewNonValidatorServer(log *logging.Logger,
	config Config,
	vegaPaths paths.Paths,
	protocolUpgradeService ProtocolUpgradeService,
) (*Server, error) {
	// setup logger
	log = log.Named(namedLogger)
	log.SetLevel(config.Level.Get())

	return &Server{
		log:                    log,
		cfg:                    config,
		nodeWallet:             nil,
		srv:                    nil,
		protocolUpgradeService: NewProtocolUpgradeService(protocolUpgradeService),
	}, nil
}

// NewServer returns a new instance of the RPC socket server.
func NewValidatorServer(
	log *logging.Logger,
	config Config,
	vegaPaths paths.Paths,
	nodeWalletPassphrase string,
	nodeWallets *nodewallets.NodeWallets,
	protocolUpgradeService ProtocolUpgradeService,
) (*Server, error) {
	// setup logger
	log = log.Named(namedLogger)
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

// Start start the server.
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
