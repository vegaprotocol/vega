package gql

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"runtime/debug"
	"strconv"
	"time"

	"code.vegaprotocol.io/data-node/gateway"
	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/data-node/metrics"
	protoapi "code.vegaprotocol.io/protos/data-node/api/v1"
	v2 "code.vegaprotocol.io/protos/data-node/api/v2"
	vegaprotoapi "code.vegaprotocol.io/protos/vega/api/v1"
	"google.golang.org/grpc"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/99designs/gqlgen/handler"
	"github.com/gorilla/websocket"
	"github.com/rs/cors"
)

const (
	namedLogger = "gateway.gql"
)

// GraphServer is the graphql server
type GraphServer struct {
	gateway.Config

	log                 *logging.Logger
	coreProxyClient     vegaprotoapi.CoreServiceClient
	tradingDataClient   protoapi.TradingDataServiceClient
	tradingDataClientV2 v2.TradingDataServiceClient
	srv                 *http.Server
}

// New returns a new instance of the grapqhl server
func New(
	log *logging.Logger,
	config gateway.Config,
) (*GraphServer, error) {
	// setup logger
	log = log.Named(namedLogger)
	log.SetLevel(config.Level.Get())

	serverAddr := fmt.Sprintf("%v:%v", config.Node.IP, config.Node.Port)

	tdconn, err := grpc.Dial(serverAddr, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	tradingDataClient := protoapi.NewTradingDataServiceClient(tdconn)
	tradingDataClientV2 := v2.NewTradingDataServiceClient(tdconn)

	tconn, err := grpc.Dial(serverAddr, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	tradingClient := vegaprotoapi.NewCoreServiceClient(tconn)

	return &GraphServer{
		log:                 log,
		Config:              config,
		coreProxyClient:     tradingClient,
		tradingDataClient:   tradingDataClient,
		tradingDataClientV2: tradingDataClientV2,
	}, nil
}

// ReloadConf update the internal configuration of the graphql server
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

// Start start the server in order receive http request
func (g *GraphServer) Start() error {
	// <--- cors support - configure for production
	corz := cors.AllowAll()
	var up = websocket.Upgrader{
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
	var config = Config{
		Resolvers: resolverRoot,
	}

	loggingMiddleware := handler.ResolverMiddleware(func(ctx context.Context, next graphql.Resolver) (res interface{}, err error) {
		resctx := graphql.GetResolverContext(ctx)
		clockstart := time.Now()
		res, err = next(ctx)
		metrics.APIRequestAndTimeGraphQL(resctx.Field.Name, time.Since(clockstart).Seconds())
		return res, err
	})

	handlr := http.NewServeMux()

	if g.GraphQLPlaygroundEnabled {
		g.log.Warn("graphql playground enabled, this is not a recommended setting for production")
		handlr.Handle("/", corz.Handler(playground.Handler("VEGA", "/query")))
	}
	options := []handler.Option{
		handler.WebsocketKeepAliveDuration(10 * time.Second),
		handler.WebsocketUpgrader(up),
		loggingMiddleware,
		handler.RecoverFunc(func(ctx context.Context, err interface{}) error {
			g.log.Warn("Recovering from error on graphQL handler",
				logging.String("error", fmt.Sprintf("%s", err)))
			debug.PrintStack()
			return errors.New("an internal error occurred")
		}),
	}
	if g.GraphQL.ComplexityLimit > 0 {
		options = append(options, handler.ComplexityLimit(g.GraphQL.ComplexityLimit))
	}
	handlr.Handle("/query", gateway.RemoteAddrMiddleware(g.log, corz.Handler(
		handler.GraphQL(NewExecutableSchema(config), options...),
	)))

	g.srv = &http.Server{
		Addr:    addr,
		Handler: handlr,
	}

	err := g.srv.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to listen and serve on graphQL server: %w", err)
	}

	return nil
}

// Stop will close the http server gracefully
func (g *GraphServer) Stop() {
	if g.srv != nil {
		g.log.Info("Stopping GraphQL based API")
		if err := g.srv.Shutdown(context.Background()); err != nil {
			g.log.Error("Failed to stop GraphQL based API cleanly",
				logging.Error(err))
		}
	}
}
