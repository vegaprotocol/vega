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
	var upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	var port = 3004
	log.Infof("Starting GraphQL based server on port %d...\n", port)
	var addr = fmt.Sprintf(":%d", port)
	var resolverRoot = NewResolverRoot(g.orderService, g.tradeService)
	http.Handle("/", handler.Playground("VEGA", "/query"))
	http.Handle("/query", cors.Default().Handler(handler.GraphQL(
		NewExecutableSchema(resolverRoot),
		handler.WebsocketUpgrader(upgrader),
		handler.RecoverFunc(func(ctx context.Context, err interface{}) error {
			log.Errorf("GraphQL error: %v", err)
			return errors.New("an error occurred from the GraphQL server, please retry")
		})),
	))

	err := http.ListenAndServe(addr, nil)
	log.Fatalf("GraphQL server fatal error: %v", err)
}
