package gql

import (
	"code.vegaprotocol.io/vega/internal/monitoring"
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"runtime/debug"
	"time"

	"code.vegaprotocol.io/vega/internal/api"
	"code.vegaprotocol.io/vega/internal/candles"
	"code.vegaprotocol.io/vega/internal/logging"
	"code.vegaprotocol.io/vega/internal/markets"
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

type graphServer struct {
	*api.Config
	timeService   vegatime.Service
	orderService  orders.Service
	tradeService  trades.Service
	candleService candles.Service
	marketService markets.Service
	partyService  parties.Service
	srv           *http.Server
	statusChecker *monitoring.Status
}

func NewGraphQLServer(
	config *api.Config,
	orderService orders.Service,
	tradeService trades.Service,
	candleService candles.Service,
	marketService markets.Service,
	partyService parties.Service,
	timeService vegatime.Service,
	statusChecker *monitoring.Status,
) *graphServer {

	return &graphServer{
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

func (g *graphServer) remoteAddrMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		logger := *g.GetLogger()
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
	logger := *g.GetLogger()

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

	logger.Info("Starting GraphQL based API", logging.String("addr", ip), logging.Int("port", port))

	addr := fmt.Sprintf("%s:%d", ip, port)
	resolverRoot := NewResolverRoot(
		g.Config,
		g.orderService,
		g.tradeService,
		g.candleService,
		g.timeService,
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
		rlogger := logger.With(logfields...)
		rlogger.Debug("GQL Start")
		start := time.Now()
		res, err = next(ctx)
		end := time.Now()
		if err != nil {
			logfields = append(logfields, logging.String("error", err.Error()))
		}
		timetaken := end.Sub(start)
		logfields = append(logfields, logging.Int64("duration_nano", timetaken.Nanoseconds()))

		rlogger = logger.With(logfields...)
		rlogger.Debug("GQL Finish")
		return res, err
	})

	handlr := http.NewServeMux()

	handlr.Handle("/", c.Handler(handler.Playground("VEGA", "/query")))
	handlr.Handle("/query", g.remoteAddrMiddleware(c.Handler(handler.GraphQL(
		NewExecutableSchema(config),
		handler.WebsocketUpgrader(up),
		loggingMiddleware,
		handler.RecoverFunc(func(ctx context.Context, err interface{}) error {
			logger.Warn("Recovering from error on graphQL handler",
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
		logger.Panic("Failed to listen and serve on graphQL server", logging.Error(err))
	}
}

func (g *graphServer) Stop() error {
	if g.srv != nil {
		return g.srv.Shutdown(context.Background())
	}
	return errors.New("Graphql server not started")
}
