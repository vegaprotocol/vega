package gql

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"runtime/debug"
	"vega/api"
	"vega/log"
	"github.com/vektah/gqlgen/handler"
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
	var port = 3004
	log.Infof("Starting GraphQL based server on port %d...\n", port)
	var addr = fmt.Sprintf(":%d", port)
	var resolverRoot = NewResolverRoot(g.orderService, g.tradeService)
	http.Handle("/", handler.Playground("Orders", "/query"))
	http.Handle("/query", handler.GraphQL(
		NewExecutableSchema(resolverRoot),
		handler.RecoverFunc(func(ctx context.Context, err interface{}) error {
			// send this panic somewhere    ß
			log.Errorf("GraphQL error: %v", err)
			debug.PrintStack()
			return errors.New("user message on panic")
		}),
	))

	err := http.ListenAndServe(addr, nil)
	log.Fatalf("GraphQL server fatal error: %v", err)
}
