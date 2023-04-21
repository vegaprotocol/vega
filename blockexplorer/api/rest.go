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
	"net/http"

	datanodeRest "code.vegaprotocol.io/vega/datanode/gateway/rest"
	"code.vegaprotocol.io/vega/logging"
	protoapi "code.vegaprotocol.io/vega/protos/blockexplorer/api/v1"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"
)

// Handler implement a rest server acting as a proxy to the grpc api.
type RESTHandler struct {
	RESTConfig
	log    *logging.Logger
	dialer grpcDialer
	mux    *runtime.ServeMux
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

func (r *RESTHandler) Start() error {
	r.log.Info("Starting REST<>GRPC based API", logging.String("endpoint", r.Endpoint))

	opts := []grpc.DialOption{grpc.WithInsecure()}
	ctx := context.Background()
	if err := r.registerBlockExplorer(ctx, r.mux, opts); err != nil {
		r.log.Panic("Failure registering trading handler for REST proxy endpoints", logging.Error(err))
	}

	return nil
}

// registerBlockExplorer is a variation of RegisterBlockExplorerHandlerFromEndpoint, which uses our custom dialer.
func (r *RESTHandler) registerBlockExplorer(ctx context.Context, mux *runtime.ServeMux, opts []grpc.DialOption) (err error) {
	conn, err := r.dialer.DialGRPC(opts...)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			if cerr := conn.Close(); cerr != nil {
				grpclog.Infof("Failed to close conn to %s: %v", r.dialer.Addr().String(), cerr)
			}
			return
		}
		go func() {
			<-ctx.Done()
			if cerr := conn.Close(); cerr != nil {
				grpclog.Infof("Failed to close conn to %s: %v", r.dialer.Addr().String(), cerr)
			}
		}()
	}()

	return protoapi.RegisterBlockExplorerServiceHandler(ctx, mux, conn)
}

func (r *RESTHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.mux.ServeHTTP(w, req)
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
