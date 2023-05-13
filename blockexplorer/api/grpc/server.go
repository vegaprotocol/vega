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
}

func NewServer(cfg Config, log *logging.Logger, blockExplorerServer pb.BlockExplorerServiceServer) *Server {
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
	}
}

func (g *Server) Serve(lis net.Listener) error {
	logAddr := logging.String("address", lis.Addr().String())
	g.log.Info("starting grpc server", logAddr)
	defer g.log.Info("stopping grpc server", logAddr)

	return g.grpc.Serve(lis)
}
