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

package api

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"code.vegaprotocol.io/vega/logging"

	"github.com/fullstorydev/grpcui/standalone"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type GRPCUIHandler struct {
	GRPCUIConfig
	handler http.Handler
	log     *logging.Logger
	dialer  grpcDialer
	conn    *grpc.ClientConn
}

type grpcDialer interface {
	net.Listener
	DialGRPC(context.Context, ...grpc.DialOption) (*grpc.ClientConn, error)
}

func NewGRPCUIHandler(log *logging.Logger, dialer grpcDialer, config GRPCUIConfig) *GRPCUIHandler {
	log = log.Named(grpcUINamedLogger)
	return &GRPCUIHandler{
		GRPCUIConfig: config,
		log:          log,
		handler:      NewNotStartedHandler("grpc-ui"),
		dialer:       dialer,
	}
}

func (g *GRPCUIHandler) Name() string {
	return "grpc-ui"
}

func (g *GRPCUIHandler) Start(ctx context.Context) error {
	defaultCallOptions := []grpc.CallOption{
		grpc.MaxCallRecvMsgSize(int(g.MaxPayloadSize)),
	}

	dialOpts := []grpc.DialOption{
		grpc.WithDefaultCallOptions(defaultCallOptions...),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	}
	conn, err := g.dialer.DialGRPC(ctx, dialOpts...)
	if err != nil {
		return fmt.Errorf("failed to create client to local grpc server: %w", err)
	}
	g.conn = conn

	g.log.Info("Starting gRPC UI", logging.String("target", conn.Target()))

	handler, err := standalone.HandlerViaReflection(ctx, conn, "vega data node")
	if err != nil {
		return fmt.Errorf("failed to create grpc-ui server:%w", err)
	}
	g.handler = handler
	return nil
}

func (g *GRPCUIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	g.handler.ServeHTTP(w, r)
}

func (g *GRPCUIHandler) Stop() {
	if g.conn != nil {
		g.log.Info("Stopping gRPC UI", logging.String("target", g.conn.Target()))
		_ = g.conn.Close()
	}
}
