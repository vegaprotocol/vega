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

	"code.vegaprotocol.io/vega/logging"
	"github.com/fullstorydev/grpcui/standalone"
	"google.golang.org/grpc"
)

type GRPCUIHandler struct {
	GRPCUIConfig
	handler http.Handler
	log     *logging.Logger
	dialer  grpcDialer
}

type grpcDialer interface {
	net.Listener
	DialGRPC(opts ...grpc.DialOption) (*grpc.ClientConn, error)
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

func (g *GRPCUIHandler) Start() error {
	defaultCallOptions := []grpc.CallOption{
		grpc.MaxCallRecvMsgSize(int(g.MaxPayloadSize)),
	}

	dialOpts := []grpc.DialOption{
		grpc.WithDefaultCallOptions(defaultCallOptions...),
		grpc.WithInsecure(),
		grpc.WithBlock(),
	}
	cc, err := g.dialer.DialGRPC(dialOpts...)
	if err != nil {
		return fmt.Errorf("failed to create client to local grpc server:%w", err)
	}
	g.log.Info("connected to grpc server", logging.String("target", cc.Target()))

	ctx := context.Background()
	handler, err := standalone.HandlerViaReflection(ctx, cc, "vega data node")
	if err != nil {
		return fmt.Errorf("failed to create grpc-ui server:%w", err)
	}
	g.handler = handler
	return nil
}

func (g *GRPCUIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	g.handler.ServeHTTP(w, r)
}
