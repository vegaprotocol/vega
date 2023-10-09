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

package api

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"github.com/fullstorydev/grpcui/standalone"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"code.vegaprotocol.io/vega/logging"
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
