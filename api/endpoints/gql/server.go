package gql

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"runtime/debug"
	"vega/api"
	"vega/log"

	"github.com/99designs/gqlgen/handler"
	"github.com/gorilla/websocket"
	"github.com/rs/cors"
)

type graphServer struct {
	orderService  api.OrderService
	tradeService  api.TradeService
	candleService api.CandleService
}

func NewGraphQLServer(orderService api.OrderService, tradeService api.TradeService, candleService api.CandleService) *graphServer {
	return &graphServer{
		orderService:  orderService,
		tradeService:  tradeService,
		candleService: candleService,
	}
}

func remoteAddrMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		found := false
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			log.Errorf("Middleware: %q is not splittable", r.RemoteAddr)
		} else {
			userIP := net.ParseIP(ip)
			if userIP == nil {
				log.Errorf("Middleware: %q is not IP:port", r.RemoteAddr)
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
	// <--- CORS support - configure for production
	var cors = cors.Default()
	var upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	// CORS support - configure for production --->
	var port = 3004
	log.Infof("Starting GraphQL based server on port %d...\n", port)
	var addr = fmt.Sprintf(":%d", port)
	var resolverRoot = NewResolverRoot(g.orderService, g.tradeService, g.candleService)
	var config = Config{
		Resolvers: resolverRoot,
	}
	http.Handle("/", cors.Handler(handler.Playground("VEGA", "/query")))
	http.Handle("/query", remoteAddrMiddleware(cors.Handler(handler.GraphQL(
		NewExecutableSchema(config),
		handler.WebsocketUpgrader(upgrader),
		handler.RecoverFunc(func(ctx context.Context, err interface{}) error {
			log.Errorf("GraphQL error: %v", err)
			debug.PrintStack()
			return errors.New("an internal error occurred")
		})),
	)))

	err := http.ListenAndServe(addr, nil)
	log.Fatalf("Fatal error with GraphQL server: %v", err)
}
