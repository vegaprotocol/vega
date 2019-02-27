package gql

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"runtime/debug"

	"vega/api"
	"vega/internal/candles"
	"vega/internal/markets"
	"vega/internal/orders"
	"vega/internal/trades"
	"vega/internal/vegatime"

	"vega/internal/logging"

	"github.com/99designs/gqlgen/handler"
	"github.com/gorilla/websocket"
	"github.com/rs/cors"
)

type graphServer struct {
	*api.Config
	timeService   vegatime.Service
	orderService  orders.Service
	tradeService  trades.Service
	candleService candles.Service
	marketService markets.Service
}

func NewGraphQLServer(config *api.Config, orderService orders.Service,
	tradeService trades.Service, candleService candles.Service, marketService markets.Service, timeService vegatime.Service) *graphServer {

	return &graphServer{
		Config:        config,
		orderService:  orderService,
		tradeService:  tradeService,
		candleService: candleService,
		timeService:   timeService,
		marketService: marketService,
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
	)
	var config = Config{
		Resolvers: resolverRoot,
	}
	http.Handle("/", c.Handler(handler.Playground("VEGA", "/query")))
	http.Handle("/query", g.remoteAddrMiddleware(c.Handler(handler.GraphQL(
		NewExecutableSchema(config),
		handler.WebsocketUpgrader(up),
		handler.RecoverFunc(func(ctx context.Context, err interface{}) error {
			logger.Warn("Recovering from error on graphQL handler",
				logging.String("error", fmt.Sprintf("%s", err)))
			debug.PrintStack()
			return errors.New("an internal error occurred")
		})),
	)))

	err := http.ListenAndServe(addr, nil)
	if err != nil {
		logger.Panic("Failed to listen and serve on graphQL server", logging.Error(err))
	}
}
