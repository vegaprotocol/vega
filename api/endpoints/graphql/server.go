package graphql

import (
	"fmt"
	"log"
	"context"
	"errors"
	"vega/api"

	"net/http"
	"github.com/vektah/gqlgen/handler"
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
	var port = 3006
	fmt.Printf("Starting GraphQL based server on port %d...\n", port)
	var addr = fmt.Sprintf(":%d", port)

	http.Handle("/", handler.Playground("Orders", "/query"))
	http.Handle("/query", handler.GraphQL(
		NewExecutableSchema(NewQueryResolver(g.orderService)),
		handler.RecoverFunc(func(ctx context.Context, err interface{}) error {
			// send this panic somewhere    ß
			log.Print(err)
			debug.PrintStack()
			return errors.New("user message on panic")
		}),
	))
	log.Fatal(http.ListenAndServe(addr, nil))
}