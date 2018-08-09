package gql

import (
	"context"
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/rs/cors"
	"github.com/vektah/gqlgen/handler"
	"net/http"
	"vega/api"
	"vega/log"
	"runtime/debug"
)

type graphServer struct {
	orderService api.OrderService
	tradeService api.TradeService
}

func NewGraphQLServer(orderService api.OrderService, tradeService api.TradeService) *graphServer {
	return &graphServer{
		orderService: orderService,
		tradeService: tradeService,
	}
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
	var resolverRoot = NewResolverRoot(g.orderService, g.tradeService)
	http.Handle("/", cors.Handler(handler.Playground("VEGA", "/query")))
	http.Handle("/query", cors.Handler(handler.GraphQL(
		NewExecutableSchema(resolverRoot),
		handler.WebsocketUpgrader(upgrader),
		handler.RecoverFunc(func(ctx context.Context, err interface{}) error {
			log.Errorf("GraphQL error: %v", err)
			debug.PrintStack()
			return errors.New("an internal error occurred")
		})),
	))

	err := http.ListenAndServe(addr, nil)
	log.Fatalf("Fatal error with GraphQL server: %v", err)
}
