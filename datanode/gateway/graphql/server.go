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

package gql

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"code.vegaprotocol.io/vega/datanode/gateway"
	"code.vegaprotocol.io/vega/datanode/metrics"
	"code.vegaprotocol.io/vega/datanode/ratelimit"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/paths"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	"code.vegaprotocol.io/vega/protos/vega"
	vegaprotoapi "code.vegaprotocol.io/vega/protos/vega/api/v1"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/99designs/gqlgen/handler"
	"github.com/gorilla/websocket"
	"github.com/vektah/gqlparser/v2/gqlerror"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const (
	namedLogger = "gql"
)

// GraphServer is the graphql server.
type GraphServer struct {
	gateway.Config

	log       *logging.Logger
	vegaPaths paths.Paths

	coreProxyClient     CoreProxyServiceClient
	tradingDataClientV2 v2.TradingDataServiceClient
	rl                  *gateway.SubscriptionRateLimiter
	rateLimit           *ratelimit.RateLimit
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

	tdconn, err := grpc.Dial(serverAddr, grpc.WithInsecure(), ratelimit.WithSecret())
	if err != nil {
		return nil, err
	}
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
		tradingDataClientV2: tradingDataClientV2,
		rl: gateway.NewSubscriptionRateLimiter(
			log, config.MaxSubscriptionPerClient),
		rateLimit: ratelimit.NewFromConfig(&config.RateLimit, log),
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

	// TODO(): not updating the actual server for now, may need to look at this later
	// e.g restart the http server on another port or whatever
	g.Config = cfg
	g.rateLimit.ReloadConfig(&cfg.RateLimit)
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

// Start starts the server in order receive http request.
func (g *GraphServer) Start() (http.Handler, error) {
	up := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	resolverRoot := NewResolverRoot(
		g.log,
		g.Config,
		g.coreProxyClient,
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
		if ctx.Value(metadataKey{}) != nil {
			res, err = next(ctx)
			return
		}

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

	errMiddleware := handler.ErrorPresenter(func(ctx context.Context, e error) *gqlerror.Error {
		if e == nil {
			return nil
		}

		st, ok := status.FromError(errors.Unwrap(e))
		if !ok {
			return graphql.DefaultErrorPresenter(ctx, e)
		}

		errsStr := []string{}
		for _, v := range st.Details() {
			v, ok := v.(*vega.ErrorDetail)
			if !ok {
				continue
			}
			errsStr = append(errsStr, v.Message)
		}

		ge := graphql.DefaultErrorPresenter(
			ctx, errors.New(strings.Join(errsStr, ", ")))
		ge.Extensions = map[string]interface{}{
			"code": st.Code(),
			"type": st.Code().String(),
		}

		return ge
	})

	handlr := http.NewServeMux()

	if g.GraphQLPlaygroundEnabled {
		g.log.Warn("graphql playground enabled, this is not a recommended setting for production")
		handlr.Handle("/", playground.Handler("VEGA", g.GraphQL.Endpoint))
	}
	options := []handler.Option{
		handler.WebsocketKeepAliveDuration(10 * time.Second),
		handler.WebsocketUpgrader(up),
		loggingMiddleware,
		headersMiddleware,
		errMiddleware,
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

	middleware := gateway.Chain(
		gateway.RemoteAddrMiddleware(g.log, handler.GraphQL(NewExecutableSchema(config), options...)),
		gateway.WithAddHeadersMiddleware,
		g.rl.WithSubscriptionRateLimiter,
		g.rateLimit.HTTPMiddleware,
	)

	handlr.Handle(g.GraphQL.Endpoint, middleware)
	return handlr, nil
}
