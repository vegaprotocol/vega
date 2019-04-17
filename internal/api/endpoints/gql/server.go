package gql

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"runtime/debug"

	"code.vegaprotocol.io/vega/internal/api"
	"code.vegaprotocol.io/vega/internal/candles"
	"code.vegaprotocol.io/vega/internal/logging"
	"code.vegaprotocol.io/vega/internal/markets"
	"code.vegaprotocol.io/vega/internal/monitoring"
	"code.vegaprotocol.io/vega/internal/orders"
	"code.vegaprotocol.io/vega/internal/parties"
	"code.vegaprotocol.io/vega/internal/trades"
	"code.vegaprotocol.io/vega/internal/vegatime"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/handler"
	"github.com/gorilla/websocket"
	"github.com/rs/cors"
	"go.uber.org/zap"
)

const (
	namedLogger = "api.gql"
)

type graphServer struct {
	api.Config

	log           *logging.Logger
	orderService  *orders.Svc
	tradeService  *trades.Svc
	candleService *candles.Svc
	marketService *markets.Svc
	partyService  *parties.Svc
	timeService   *vegatime.Svc
	srv           *http.Server
	statusChecker *monitoring.Status
}

func NewGraphQLServer(
	log *logging.Logger,
	config api.Config,
	orderService *orders.Svc,
	tradeService *trades.Svc,
	candleService *candles.Svc,
	marketService *markets.Svc,
	partyService *parties.Svc,
	timeService *vegatime.Svc,
	statusChecker *monitoring.Status,
) *graphServer {
	// setup logger
	log = log.Named(namedLogger)
	log.SetLevel(config.Level.Get())

	return &graphServer{
		log:           log,
		Config:        config,
		orderService:  orderService,
		tradeService:  tradeService,
		candleService: candleService,
		timeService:   timeService,
		marketService: marketService,
		partyService:  partyService,
		statusChecker: statusChecker,
	}
}

func (s *graphServer) ReloadConf(cfg api.Config) {
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

func (g *graphServer) remoteAddrMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		logger := g.log
		found := false
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			logger.Warn("Remote address is not splittable in middleware",
				logging.String("remote-addr", r.RemoteAddr))
		} else {
			userIP := net.ParseIP(ip)
			if userIP == nil {
				logger.Warn("Remote address is not IP:port format in middleware",
					logging.String("remote-addr", r.RemoteAddr))
			} else {
				found = true

				// Only defined when site is accessed via non-anonymous proxy
				// and takes precedence over RemoteAddr
				forward := r.Header.Get("X-Forwarded-For")
				if forward != "" {
					ip = forward
				}
			}
		}

		if found {
			ctx := context.WithValue(r.Context(), "remote-ip-addr", ip)
			next.ServeHTTP(w, r.WithContext(ctx))
		} else {
			next.ServeHTTP(w, r)
		}
	})
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

	port := g.GraphQLServerPort
	ip := g.GraphQLServerIpAddress

	g.log.Info("Starting GraphQL based API", logging.String("addr", ip), logging.Int("port", port))

	addr := fmt.Sprintf("%s:%d", ip, port)
	resolverRoot := NewResolverRoot(
		g.log,
		g.Config,
		g.orderService,
		g.tradeService,
		g.candleService,
		g.marketService,
		g.partyService,
		g.statusChecker,
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
	handlr.Handle("/query", api.RemoteAddrMiddleware(logger, c.Handler(handler.GraphQL(
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
