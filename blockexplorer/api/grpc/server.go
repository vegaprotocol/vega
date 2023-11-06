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

package grpc

import (
	"net"

	"code.vegaprotocol.io/vega/logging"
	pb "code.vegaprotocol.io/vega/protos/blockexplorer/api/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type Server struct {
	Config
	log           *logging.Logger
	blockExplorer pb.BlockExplorerServiceServer
	grpc          *grpc.Server
	lis           net.Listener
}

func NewServer(cfg Config, log *logging.Logger, blockExplorerServer pb.BlockExplorerServiceServer, lis net.Listener) *Server {
	log = log.Named(namedLogger)

	grpcServer := grpc.NewServer()
	pb.RegisterBlockExplorerServiceServer(grpcServer, blockExplorerServer)
	if cfg.Reflection {
		reflection.Register(grpcServer)
	}

	return &Server{
		Config:        cfg,
		log:           log,
		blockExplorer: blockExplorerServer,
		grpc:          grpcServer,
		lis:           lis,
	}
}

func (g *Server) Serve() error {
	g.log.Info("Starting gRPC server", logging.String("address", g.lis.Addr().String()))
	return g.grpc.Serve(g.lis)
}

func (g *Server) Stop() {
	if g.grpc != nil {
		g.log.Info("Stopping gRPC server", logging.String("address", g.lis.Addr().String()))
		g.grpc.Stop()
	}
}
