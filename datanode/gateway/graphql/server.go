// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.DATANODE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package gql

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"runtime/debug"
	"strconv"
	"time"

	"code.vegaprotocol.io/vega/datanode/gateway"
	"code.vegaprotocol.io/vega/datanode/metrics"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/paths"
	protoapi "code.vegaprotocol.io/vega/protos/data-node/api/v1"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	vegaprotoapi "code.vegaprotocol.io/vega/protos/vega/api/v1"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/99designs/gqlgen/handler"
	"github.com/gorilla/websocket"
	"github.com/rs/cors"
	"golang.org/x/crypto/acme/autocert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

const (
	namedLogger = "gateway.gql"
)

// GraphServer is the graphql server.
type GraphServer struct {
	gateway.Config

	log       *logging.Logger
	vegaPaths paths.Paths

	coreProxyClient     CoreProxyServiceClient
	tradingDataClient   protoapi.TradingDataServiceClient
	tradingDataClientV2 v2.TradingDataServiceClient
	srv                 *http.Server
}

// New returns a new instance of the grapqhl server.
func New(
	log *logging.Logger,
	config gateway.Config,
	vegaPaths paths.Paths,
) (*GraphServer, error) {
	// setup logger
	log = log.Named(namedLogger)
	log.SetLevel(config.Level.Get())

	serverAddr := fmt.Sprintf("%v:%v", config.Node.IP, config.Node.Port)

	tdconn, err := grpc.Dial(serverAddr, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	tradingDataClient := protoapi.NewTradingDataServiceClient(&clientConn{tdconn})
	tradingDataClientV2 := v2.NewTradingDataServiceClient(&clientConn{tdconn})

	tconn, err := grpc.Dial(serverAddr, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}

	tradingClient := vegaprotoapi.NewCoreServiceClient(&clientConn{tconn})

	return &GraphServer{
		log:                 log,
		Config:              config,
		vegaPaths:           vegaPaths,
		coreProxyClient:     tradingClient,
		tradingDataClient:   tradingDataClient,
		tradingDataClientV2: tradingDataClientV2,
	}, nil
}

// ReloadConf update the internal configuration of the graphql server.
func (g *GraphServer) ReloadConf(cfg gateway.Config) {
	g.log.Info("reloading configuration")
	if g.log.GetLevel() != cfg.Level.Get() {
		g.log.Info("updating log level",
			logging.String("old", g.log.GetLevel().String()),
			logging.String("new", cfg.Level.String()),
		)
		g.log.SetLevel(cfg.Level.Get())
	}

	// TODO(): not updating the the actual server for now, may need to look at this later
	// e.g restart the http server on another port or whatever
	g.Config = cfg
}

type (
	clientConn struct {
		*grpc.ClientConn
	}
	metadataKey struct{}
)

func (c *clientConn) Invoke(ctx context.Context, method string, args, reply interface{}, opts ...grpc.CallOption) error {
	mdi := ctx.Value(metadataKey{})
	if md, ok := mdi.(*metadata.MD); ok {
		opts = append(opts, grpc.Header(md))
	}
	return c.ClientConn.Invoke(ctx, method, args, reply, opts...)
}

// Start start the server in order receive http request.
func (g *GraphServer) Start() error {
	// <--- cors support - configure for production
	corz := cors.AllowAll()
	up := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	// cors support - configure for production --->

	port := g.GraphQL.Port
	ip := g.GraphQL.IP

	g.log.Info("Starting GraphQL based API", logging.String("addr", ip), logging.Int("port", port))

	addr := net.JoinHostPort(ip, strconv.Itoa(port))
	resolverRoot := NewResolverRoot(
		g.log,
		g.Config,
		g.coreProxyClient,
		g.tradingDataClient,
		g.tradingDataClientV2,
	)
	config := Config{
		Resolvers: resolverRoot,
	}

	loggingMiddleware := handler.ResolverMiddleware(func(ctx context.Context, next graphql.Resolver) (res interface{}, err error) {
		resctx := graphql.GetResolverContext(ctx)
		clockstart := time.Now()
		res, err = next(ctx)
		metrics.APIRequestAndTimeGraphQL(resctx.Field.Name, time.Since(clockstart).Seconds())
		return res, err
	})

	headersMiddleware := handler.ResolverMiddleware(func(ctx context.Context, next graphql.Resolver) (res interface{}, err error) {
		md := metadata.MD{}
		ctx = context.WithValue(ctx, metadataKey{}, &md)
		res, err = next(ctx)
		rw, ok := gateway.InjectableWriterFromContext(ctx)
		if !ok {
			return
		}
		rw.SetHeaders(http.Header(md))
		return
	})

	handlr := http.NewServeMux()

	if g.GraphQLPlaygroundEnabled {
		g.log.Warn("graphql playground enabled, this is not a recommended setting for production")
		handlr.Handle("/", corz.Handler(playground.Handler("VEGA", g.GraphQL.Endpoint)))
	}
	options := []handler.Option{
		handler.WebsocketKeepAliveDuration(10 * time.Second),
		handler.WebsocketUpgrader(up),
		loggingMiddleware,
		headersMiddleware,
		handler.RecoverFunc(func(ctx context.Context, err interface{}) error {
			g.log.Warn("Recovering from error on graphQL handler",
				logging.String("error", fmt.Sprintf("%s", err)))
			debug.PrintStack()
			return errors.New("an internal error occurred")
		}),

		handler.ComplexityLimit(3750),
	}
	if g.GraphQL.ComplexityLimit > 0 {
		options = append(options, handler.ComplexityLimit(g.GraphQL.ComplexityLimit))
	}

	// FIXME(jeremy): to be removed once everyone has move to the new endpoint
	middleware := corz.Handler(gateway.Chain(
		gateway.RemoteAddrMiddleware(g.log, handler.GraphQL(NewExecutableSchema(config), options...)),
		gateway.WithAddHeadersMiddleware,
	))

	handlr.Handle("/query", middleware)
	handlr.Handle(g.GraphQL.Endpoint, middleware)

	// Set up https if we are using it
	var tlsConfig *tls.Config

	var cert, key string
	if g.GraphQL.HTTPSEnabled {
		if g.GraphQL.CertificateFile != "" {
			cert = g.GraphQL.CertificateFile
		}
		if g.GraphQL.KeyFile != "" {
			key = g.GraphQL.KeyFile
		}

		if g.GraphQL.AutoCertDomain != "" {
			dataNodeHome := paths.StatePath(g.vegaPaths.StatePathFor(paths.DataNodeStateHome))
			certDir := paths.JoinStatePath(dataNodeHome, "graphql_https_certificates")

			certManager := autocert.Manager{
				Prompt:     autocert.AcceptTOS,
				HostPolicy: autocert.HostWhitelist(g.GraphQL.AutoCertDomain),
				Cache:      autocert.DirCache(certDir),
			}
			tlsConfig = &tls.Config{
				GetCertificate: certManager.GetCertificate,
				NextProtos:     []string{"http/1.1", "acme-tls/1"},
			}
		}
	} else {
		g.log.Warn("GraphQL server is not configured to use HTTPS, which is required for subscriptions to work. Please see README.md for help configuring")
	}

	g.srv = &http.Server{
		Addr:      addr,
		Handler:   handlr,
		TLSConfig: tlsConfig,
	}

	var err error
	if g.GraphQL.HTTPSEnabled {
		err = g.srv.ListenAndServeTLS(cert, key)
	} else {
		err = g.srv.ListenAndServe()
	}
	if err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to listen and serve on graphQL server: %w", err)
	}

	return nil
}

// Stop will close the http server gracefully.
func (g *GraphServer) Stop() {
	if g.srv != nil {
		g.log.Info("Stopping GraphQL based API")
		if err := g.srv.Shutdown(context.Background()); err != nil {
			g.log.Error("Failed to stop GraphQL based API cleanly",
				logging.Error(err))
		}
	}
}
