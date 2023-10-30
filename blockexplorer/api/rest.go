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
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	datanodeRest "code.vegaprotocol.io/vega/datanode/gateway/rest"
	"code.vegaprotocol.io/vega/logging"
	protoapi "code.vegaprotocol.io/vega/protos/blockexplorer/api/v1"
)

// Handler implement a rest server acting as a proxy to the grpc api.
type RESTHandler struct {
	RESTConfig
	log    *logging.Logger
	dialer grpcDialer
	mux    *runtime.ServeMux
	conn   *grpc.ClientConn
}

func NewRESTHandler(log *logging.Logger, dialer grpcDialer, config RESTConfig) *RESTHandler {
	log = log.Named(restNamedLogger)
	log.SetLevel(config.Level.Get())

	return &RESTHandler{
		log:        log,
		RESTConfig: config,
		dialer:     dialer,
		mux:        runtime.NewServeMux(restHandlerServeMuxOptions()...),
	}
}

func (r *RESTHandler) Name() string { return "REST" }

func (r *RESTHandler) Start(ctx context.Context) error {
	r.log.Info("Starting REST API", logging.String("endpoint", r.Endpoint))

	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	if err := r.registerBlockExplorer(ctx, r.mux, opts); err != nil {
		r.log.Panic("Failure registering trading handler for REST proxy endpoints", logging.Error(err))
	}

	return nil
}

// registerBlockExplorer is a variation of RegisterBlockExplorerHandlerFromEndpoint, which uses our custom dialer.
func (r *RESTHandler) registerBlockExplorer(ctx context.Context, mux *runtime.ServeMux, opts []grpc.DialOption) error {
	conn, err := r.dialer.DialGRPC(ctx, opts...)
	if err != nil {
		return err
	}
	r.conn = conn
	return protoapi.RegisterBlockExplorerServiceHandler(ctx, mux, conn)
}

func (r *RESTHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.mux.ServeHTTP(w, req)
}

func (r *RESTHandler) Stop() {
	if r.conn != nil {
		r.log.Info("Stopping REST API")
		_ = r.conn.Close()
	}
}

func restHandlerServeMuxOptions() []runtime.ServeMuxOption {
	jsonPB := &datanodeRest.JSONPb{
		EmitDefaults: true,
		Indent:       "  ", // formatted json output
		OrigName:     false,
	}

	return []runtime.ServeMuxOption{
		runtime.WithMarshalerOption(runtime.MIMEWildcard, jsonPB),
	}
}
