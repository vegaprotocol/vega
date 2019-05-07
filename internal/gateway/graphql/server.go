package gql

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"runtime/debug"

	"code.vegaprotocol.io/vega/internal/gateway"
	"code.vegaprotocol.io/vega/internal/logging"
	"code.vegaprotocol.io/vega/internal/vegatime"
	"code.vegaprotocol.io/vega/proto/api"
	protoapi "code.vegaprotocol.io/vega/proto/api"
	"google.golang.org/grpc"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/handler"
	"github.com/gorilla/websocket"
	"github.com/rs/cors"
	"go.uber.org/zap"
)

const (
	namedLogger = "gateway.gql"
)

type graphServer struct {
	gateway.Config

	log               *logging.Logger
	tradingClient     protoapi.TradingClient
	tradingDataClient protoapi.TradingDataClient
	srv               *http.Server
}

func New(
	log *logging.Logger,
	config gateway.Config,
) (*graphServer, error) {
	// setup logger
	log = log.Named(namedLogger)
	log.SetLevel(config.Level.Get())

	serverAddr := fmt.Sprintf("%v:%v", config.Node.IP, config.Node.Port)

	tdconn, err := grpc.Dial(serverAddr, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	tradingDataClient := api.NewTradingDataClient(tdconn)

	tconn, err := grpc.Dial(serverAddr, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	tradingClient := api.NewTradingClient(tconn)

	return &graphServer{
		log:               log,
		Config:            config,
		tradingClient:     tradingClient,
		tradingDataClient: tradingDataClient,
	}, nil
}

func (s *graphServer) ReloadConf(cfg gateway.Config) {
	s.log.Info("reloading confioguration")
	if s.log.GetLevel() != cfg.Level.Get() {
		s.log.Info("updating log level",
			logging.String("old", s.log.GetLevel().String()),
			logging.String("new", cfg.Level.String()),
		)
		s.log.SetLevel(cfg.Level.Get())
	}

	// TODO(): not updating the the actual server for now, may need to look at this later
	// e.g restart the http server on another port or whatever
	s.Config = cfg
}

func (g *graphServer) Start() {
	// <--- cors support - configure for production
	var c = cors.Default()
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

	addr := fmt.Sprintf("%s:%d", ip, port)
	resolverRoot := NewResolverRoot(
		g.log,
		g.Config,
		g.tradingClient,
		g.tradingDataClient,
	)
	var config = Config{
		Resolvers: resolverRoot,
	}

	loggingMiddleware := handler.ResolverMiddleware(func(ctx context.Context, next graphql.Resolver) (res interface{}, err error) {
		reqctx := graphql.GetRequestContext(ctx)
		logfields := make([]zap.Field, 0)
		logfields = append(logfields, logging.String("raw", reqctx.RawQuery))
		rlogger := g.log.With(logfields...)
		rlogger.Debug("GQL Start")
		start := vegatime.Now()
		res, err = next(ctx)
		end := vegatime.Now()
		if err != nil {
			logfields = append(logfields, logging.String("error", err.Error()))
		}
		timetaken := end.Sub(start)
		logfields = append(logfields, logging.Int64("duration_nano", timetaken.Nanoseconds()))

		rlogger = g.log.With(logfields...)
		rlogger.Debug("GQL Finish")
		return res, err
	})

	handlr := http.NewServeMux()

	handlr.Handle("/", c.Handler(handler.Playground("VEGA", "/query")))
	handlr.Handle("/query", gateway.RemoteAddrMiddleware(g.log, c.Handler(handler.GraphQL(
		NewExecutableSchema(config),
		handler.WebsocketUpgrader(up),
		loggingMiddleware,
		handler.RecoverFunc(func(ctx context.Context, err interface{}) error {
			g.log.Warn("Recovering from error on graphQL handler",
				logging.String("error", fmt.Sprintf("%s", err)))
			debug.PrintStack()
			return errors.New("an internal error occurred")
		})),
	)))

	g.srv = &http.Server{
		Addr:    addr,
		Handler: handlr,
	}

	err := g.srv.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		g.log.Panic("Failed to listen and serve on graphQL server", logging.Error(err))
	}
}

func (g *graphServer) Stop() {
	if g.srv != nil {
		g.log.Info("Stopping GraphQL based API")
		if err := g.srv.Shutdown(context.Background()); err != nil {
			g.log.Error("Failed to stop GraphQL based API cleanly",
				logging.Error(err))
		}
	}
}
