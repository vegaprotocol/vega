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

package blockchain

import "code.vegaprotocol.io/vega/logging"

type ChainServerImpl interface {
	ReloadConf(cfg Config)
	Stop() error
	Start() error
}

// Server abstraction for the abci server.
type Server struct {
	*Config
	log *logging.Logger
	srv ChainServerImpl
}

// NewServer instantiate a new blockchain server.
func NewServer(log *logging.Logger, srv ChainServerImpl) *Server {
	return &Server{
		log: log,
		srv: srv,
	}
}

func (s *Server) Start() error {
	return s.srv.Start()
}

// Stop gracefully shutdowns down the blockchain provider's server.
func (s *Server) Stop() error {
	s.log.Info("Stopping blockchain server")

	return s.srv.Stop()
}

func (s *Server) ReloadConf(cfg Config) {
	s.srv.ReloadConf(cfg)
}
