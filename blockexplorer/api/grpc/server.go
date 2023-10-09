// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package grpc

import (
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"code.vegaprotocol.io/vega/logging"
	pb "code.vegaprotocol.io/vega/protos/blockexplorer/api/v1"
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
