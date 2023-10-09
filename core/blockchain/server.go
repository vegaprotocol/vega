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
